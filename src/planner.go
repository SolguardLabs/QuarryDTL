package main

import "sort"

type PlannerOptions struct {
	UseProjected bool
	MinimumBatch Amount
}

type LiquidityView struct {
	Profiles []CapacityProfile `json:"profiles"`
	Clock    int64             `json:"clock"`
}

func (v LiquidityView) ByVault(id string) (CapacityProfile, bool) {
	for _, profile := range v.Profiles {
		if profile.VaultID == id {
			return profile, true
		}
	}
	return CapacityProfile{}, false
}

func (v LiquidityView) TotalAssignable(asset string) Amount {
	total := Amount(0)
	for _, profile := range v.Profiles {
		if profile.Asset == asset {
			total += profile.Assignable
		}
	}
	return total
}

type ViewBuilder struct {
	ledger *Ledger
}

func NewViewBuilder(ledger *Ledger) ViewBuilder {
	return ViewBuilder{ledger: ledger}
}

func (b ViewBuilder) Build(options PlannerOptions) LiquidityView {
	profiles := make([]CapacityProfile, 0)
	for _, vault := range b.ledger.Vaults.List() {
		profiles = append(profiles, b.ProfileForVault(vault.ID, options))
	}
	sort.SliceStable(profiles, func(i, j int) bool {
		if profiles[i].Weight() == profiles[j].Weight() {
			return profiles[i].VaultID < profiles[j].VaultID
		}
		return profiles[i].Weight() > profiles[j].Weight()
	})
	return LiquidityView{Profiles: profiles, Clock: b.ledger.Clock}
}

func (b ViewBuilder) ProfileForVault(vaultID string, options PlannerOptions) CapacityProfile {
	vault := b.ledger.Vaults.MustGet(vaultID)
	minBuffer := b.ledger.Policy.MinBuffer(vault)
	free := vault.Idle - minBuffer
	if free < 0 {
		free = 0
	}
	forecast := Amount(0)
	confidence := int64(10_000)
	mode := CapacityObserved
	if options.UseProjected && b.ledger.Policy.ProjectedCapacityEnabled {
		for _, item := range b.ledger.Forecasts.AssignableForVault(vault.ID, b.ledger.Clock, b.ledger.Policy.ForecastConfidenceBps) {
			forecast += item.Amount
			if item.ConfidenceBps < confidence {
				confidence = item.ConfidenceBps
			}
		}
		if forecast > 0 {
			mode = CapacityProjected
		}
	}
	assignable := free + forecast
	if options.MinimumBatch > 0 && assignable < options.MinimumBatch {
		assignable = 0
	}
	return CapacityProfile{
		VaultID:       vault.ID,
		Asset:         vault.Asset,
		Mode:          mode,
		Free:          free,
		Reserved:      vault.Reserved,
		InFlight:      vault.InFlight,
		PendingIn:     vault.PendingIn,
		PendingOut:    vault.PendingOut,
		Forecast:      forecast,
		Assignable:    assignable,
		MinBuffer:     minBuffer,
		ConfidenceBps: confidence,
	}
}

func (b ViewBuilder) RouteCapacity(route *Route, options PlannerOptions) CapacityProfile {
	profile := b.ProfileForVault(route.SourceVault, options)
	vault := b.ledger.Vaults.MustGet(route.SourceVault)
	base := vault.Reserve
	if profile.UsesProjection() {
		base += profile.Forecast
	}
	routeCap := base.MulBps(b.ledger.Policy.MaxRouteShareBps)
	if route.MaxAllocation > 0 {
		remainingCap := route.MaxAllocation - route.Allocated
		if routeCap > remainingCap {
			routeCap = remainingCap
		}
	}
	if profile.Assignable > routeCap {
		profile.Assignable = routeCap
	}
	if profile.Assignable > route.RemainingDemand() {
		profile.Assignable = route.RemainingDemand()
	}
	if profile.Assignable < route.MinBatch {
		profile.Assignable = 0
	}
	return profile
}

func (b ViewBuilder) RankedRoutes(options PlannerOptions) []RouteCapacity {
	candidates := []RouteCapacity{}
	for _, route := range b.ledger.Routes.OpenRoutes() {
		profile := b.RouteCapacity(route, options)
		if profile.Assignable <= 0 {
			continue
		}
		candidates = append(candidates, RouteCapacity{
			RouteID:  route.ID,
			VaultID:  route.SourceVault,
			Asset:    route.Asset,
			Capacity: profile,
			Priority: route.Priority,
			Demand:   route.RemainingDemand(),
			Score:    scoreRoute(route, profile),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].RouteID < candidates[j].RouteID
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates
}

type RouteCapacity struct {
	RouteID  string          `json:"route_id"`
	VaultID  string          `json:"vault_id"`
	Asset    string          `json:"asset"`
	Capacity CapacityProfile `json:"capacity"`
	Priority int             `json:"priority"`
	Demand   Amount          `json:"demand"`
	Score    int64           `json:"score"`
}

func scoreRoute(route *Route, profile CapacityProfile) int64 {
	score := int64(route.Priority) * 100_000
	score += int64(route.HealthBps)
	score += profile.Weight() / 1_000
	if profile.UsesProjection() {
		score += 1_500
	}
	return score
}

func splitCapacity(amount Amount, maxChunks int) []Amount {
	if amount <= 0 {
		return nil
	}
	if maxChunks <= 1 {
		return []Amount{amount}
	}
	chunk := amount / Amount(maxChunks)
	if chunk <= 0 {
		return []Amount{amount}
	}
	out := make([]Amount, 0, maxChunks)
	remaining := amount
	for i := 0; i < maxChunks; i++ {
		part := chunk
		if i == maxChunks-1 {
			part = remaining
		}
		out = append(out, part)
		remaining -= part
		if remaining <= 0 {
			break
		}
	}
	return out
}
