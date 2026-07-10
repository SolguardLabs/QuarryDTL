package main

import (
	"fmt"
	"sort"
)

type RouteStatus string

const (
	RouteOpen      RouteStatus = "open"
	RouteThrottled RouteStatus = "throttled"
	RouteClosed    RouteStatus = "closed"
)

type Route struct {
	ID              string      `json:"id"`
	SourceVault     string      `json:"source_vault"`
	TargetVault     string      `json:"target_vault"`
	Asset           string      `json:"asset"`
	Priority        int         `json:"priority"`
	Demand          Amount      `json:"demand"`
	Outstanding     Amount      `json:"outstanding"`
	Allocated       Amount      `json:"allocated"`
	Settled         Amount      `json:"settled"`
	Failed          Amount      `json:"failed"`
	MaxAllocation   Amount      `json:"max_allocation"`
	MinBatch        Amount      `json:"min_batch"`
	FeeBps          int64       `json:"fee_bps"`
	HealthBps       int64       `json:"health_bps"`
	ExpectedLatency int64       `json:"expected_latency"`
	Status          RouteStatus `json:"status"`
	Tags            LabelSet    `json:"tags,omitempty"`
}

func NewRoute(id, sourceVault, targetVault, asset string, demand Amount) *Route {
	return &Route{
		ID:              id,
		SourceVault:     sourceVault,
		TargetVault:     targetVault,
		Asset:           asset,
		Priority:        1,
		Demand:          demand,
		MaxAllocation:   demand,
		MinBatch:        1,
		FeeBps:          3,
		HealthBps:       10_000,
		ExpectedLatency: 2,
		Status:          RouteOpen,
		Tags:            LabelSet{},
	}
}

func (r *Route) Validate() error {
	if r.ID == "" || r.SourceVault == "" || r.TargetVault == "" || r.Asset == "" {
		return fmt.Errorf("route missing required fields")
	}
	if r.SourceVault == r.TargetVault {
		return fmt.Errorf("route %s must move across different vaults", r.ID)
	}
	if r.Demand < 0 || r.Outstanding < 0 || r.Allocated < 0 {
		return fmt.Errorf("route %s has negative accounting", r.ID)
	}
	if r.MaxAllocation <= 0 {
		return fmt.Errorf("route %s max allocation must be positive", r.ID)
	}
	return nil
}

func (r *Route) IsOpen() bool {
	return r.Status == RouteOpen
}

func (r *Route) RemainingDemand() Amount {
	remaining := r.Demand - r.Allocated
	if remaining < 0 {
		return 0
	}
	if r.MaxAllocation > 0 && remaining > r.MaxAllocation-r.Allocated {
		remaining = r.MaxAllocation - r.Allocated
	}
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (r *Route) Fee(amount Amount) Amount {
	return amount.MulBps(r.FeeBps)
}

func (r *Route) RegisterAllocation(amount Amount) {
	r.Allocated += amount
	r.Outstanding += amount
}

func (r *Route) RegisterSettlement(amount Amount) {
	if r.Outstanding >= amount {
		r.Outstanding -= amount
	} else {
		r.Outstanding = 0
	}
	r.Settled += amount
}

func (r *Route) RegisterFailure(amount Amount) {
	if r.Outstanding >= amount {
		r.Outstanding -= amount
	} else {
		r.Outstanding = 0
	}
	r.Failed += amount
}

func (r *Route) Snapshot() Route {
	copy := *r
	copy.Tags = r.Tags.Clone()
	return copy
}

type RouteBook struct {
	routes map[string]*Route
	order  []string
}

func NewRouteBook() *RouteBook {
	return &RouteBook{routes: map[string]*Route{}, order: []string{}}
}

func (b *RouteBook) Add(route *Route) error {
	if route == nil {
		return fmt.Errorf("route missing")
	}
	if err := route.Validate(); err != nil {
		return err
	}
	if _, ok := b.routes[route.ID]; !ok {
		b.order = append(b.order, route.ID)
		b.routes[route.ID] = route
		sort.SliceStable(b.order, func(i, j int) bool {
			left := b.routes[b.order[i]]
			right := b.routes[b.order[j]]
			if left.Priority == right.Priority {
				return left.ID < right.ID
			}
			return left.Priority > right.Priority
		})
	} else {
		b.routes[route.ID] = route
	}
	return nil
}

func (b *RouteBook) MustAdd(route *Route) {
	if err := b.Add(route); err != nil {
		panic(err)
	}
}

func (b *RouteBook) Get(id string) (*Route, bool) {
	route, ok := b.routes[id]
	return route, ok
}

func (b *RouteBook) MustGet(id string) *Route {
	route, ok := b.Get(id)
	if !ok {
		panic("missing route " + id)
	}
	return route
}

func (b *RouteBook) List() []Route {
	out := make([]Route, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, b.routes[id].Snapshot())
	}
	return out
}

func (b *RouteBook) OpenRoutes() []*Route {
	out := make([]*Route, 0, len(b.routes))
	for _, id := range b.order {
		route := b.routes[id]
		if route.IsOpen() && route.RemainingDemand() > 0 {
			out = append(out, route)
		}
	}
	return out
}

func (b *RouteBook) TotalDemand() Amount {
	total := Amount(0)
	for _, route := range b.routes {
		total += route.Demand
	}
	return total
}

func (b *RouteBook) TotalOutstanding() Amount {
	total := Amount(0)
	for _, route := range b.routes {
		total += route.Outstanding
	}
	return total
}
