package main

type InvariantReport struct {
	VaultsNonNegative        bool `json:"vaults_non_negative"`
	AccountsNonNegative      bool `json:"accounts_non_negative"`
	SettlementLinksValid     bool `json:"settlement_links_valid"`
	ForecastLinksValid       bool `json:"forecast_links_valid"`
	RoutesWithinDemand       bool `json:"routes_within_demand"`
	CapacityViewsNonNegative bool `json:"capacity_views_non_negative"`
	WithdrawalsOpen          bool `json:"withdrawals_open"`
}

type RiskMetrics struct {
	OpenSettlements       int    `json:"open_settlements"`
	FinalizedSettlements  int    `json:"finalized_settlements"`
	FailedSettlements     int    `json:"failed_settlements"`
	SubmittedRebalances   int    `json:"submitted_rebalances"`
	TotalDemand           Amount `json:"total_demand"`
	TotalOutstanding      Amount `json:"total_outstanding"`
	ProjectedCapacity     Amount `json:"projected_capacity"`
	ObservedCapacity      Amount `json:"observed_capacity"`
	LiquidationRecoveries Amount `json:"liquidation_recoveries"`
	ProtocolFeesCollected Amount `json:"protocol_fees_collected"`
}

type RiskEngine struct {
	ledger *Ledger
}

func NewRiskEngine(ledger *Ledger) RiskEngine {
	return RiskEngine{ledger: ledger}
}

func (r RiskEngine) Invariants() InvariantReport {
	return InvariantReport{
		VaultsNonNegative:        r.ledger.Vaults.AllNonNegative(),
		AccountsNonNegative:      r.ledger.Accounts.AllNonNegative(),
		SettlementLinksValid:     r.settlementLinksValid(),
		ForecastLinksValid:       r.forecastLinksValid(),
		RoutesWithinDemand:       r.routesWithinDemand(),
		CapacityViewsNonNegative: r.capacityViewsNonNegative(),
		WithdrawalsOpen:          r.withdrawalsOpen(),
	}
}

func (r RiskEngine) Metrics() RiskMetrics {
	observed := NewViewBuilder(r.ledger).Build(PlannerOptions{UseProjected: false})
	projected := NewViewBuilder(r.ledger).Build(PlannerOptions{UseProjected: true})
	return RiskMetrics{
		OpenSettlements:       r.countSettlements(SettlementOpen) + r.countSettlements(SettlementPrepared) + r.countSettlements(SettlementSubmitted),
		FinalizedSettlements:  r.countSettlements(SettlementFinalized),
		FailedSettlements:     r.countSettlements(SettlementFailed),
		SubmittedRebalances:   r.ledger.Journal.Count("rebalance.executed"),
		TotalDemand:           r.ledger.Routes.TotalDemand(),
		TotalOutstanding:      r.ledger.Routes.TotalOutstanding(),
		ProjectedCapacity:     projected.TotalAssignable("usdc") + projected.TotalAssignable("usdt"),
		ObservedCapacity:      observed.TotalAssignable("usdc") + observed.TotalAssignable("usdt"),
		LiquidationRecoveries: r.ledger.Journal.Amount("liquidation.recovered"),
		ProtocolFeesCollected: r.protocolFees(),
	}
}

func (r RiskEngine) countSettlements(status SettlementStatus) int {
	count := 0
	for _, settlement := range r.ledger.Settlements.List() {
		if settlement.Status == status {
			count++
		}
	}
	return count
}

func (r RiskEngine) protocolFees() Amount {
	total := Amount(0)
	for _, settlement := range r.ledger.Settlements.List() {
		if settlement.Status == SettlementFinalized {
			total += settlement.Fee
		}
	}
	return total
}

func (r RiskEngine) settlementLinksValid() bool {
	for _, settlement := range r.ledger.Settlements.List() {
		if _, ok := r.ledger.Routes.Get(settlement.Route); !ok {
			return false
		}
		if _, ok := r.ledger.Vaults.Get(settlement.SourceVault); !ok {
			return false
		}
		if _, ok := r.ledger.Vaults.Get(settlement.TargetVault); !ok {
			return false
		}
	}
	return true
}

func (r RiskEngine) forecastLinksValid() bool {
	for _, forecast := range r.ledger.Forecasts.List() {
		if forecast.SourceSettlement == "" {
			continue
		}
		if _, ok := r.ledger.Settlements.Get(forecast.SourceSettlement); !ok {
			return false
		}
		if _, ok := r.ledger.Vaults.Get(forecast.Vault); !ok {
			return false
		}
	}
	return true
}

func (r RiskEngine) routesWithinDemand() bool {
	for _, route := range r.ledger.Routes.List() {
		if route.Allocated > route.Demand {
			return false
		}
		if route.Settled+route.Failed > route.Allocated {
			return false
		}
		if route.Outstanding < 0 {
			return false
		}
	}
	return true
}

func (r RiskEngine) capacityViewsNonNegative() bool {
	view := NewViewBuilder(r.ledger).Build(PlannerOptions{UseProjected: true})
	for _, profile := range view.Profiles {
		if profile.Assignable < 0 || profile.Free < 0 || profile.Forecast < 0 {
			return false
		}
	}
	return true
}

func (r RiskEngine) withdrawalsOpen() bool {
	for _, vault := range r.ledger.Vaults.List() {
		if vault.Status == VaultPaused {
			return false
		}
		if vault.Reserve+vault.PendingIn < vault.PendingOut {
			return false
		}
	}
	return true
}

func (r RiskEngine) AllGreen() bool {
	invariants := r.Invariants()
	return invariants.VaultsNonNegative &&
		invariants.AccountsNonNegative &&
		invariants.SettlementLinksValid &&
		invariants.ForecastLinksValid &&
		invariants.RoutesWithinDemand &&
		invariants.CapacityViewsNonNegative &&
		invariants.WithdrawalsOpen
}
