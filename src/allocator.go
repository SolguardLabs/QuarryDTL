package main

type AllocationPlan struct {
	ID             string       `json:"id"`
	Route          string       `json:"route"`
	SourceVault    string       `json:"source_vault"`
	TargetVault    string       `json:"target_vault"`
	Asset          string       `json:"asset"`
	Amount         Amount       `json:"amount"`
	Mode           CapacityMode `json:"mode"`
	ProjectedShare Amount       `json:"projected_share"`
	Reason         string       `json:"reason"`
}

type AllocationBatch struct {
	ID        string           `json:"id"`
	Clock     int64            `json:"clock"`
	Plans     []AllocationPlan `json:"plans"`
	Requested Amount           `json:"requested"`
	Accepted  Amount           `json:"accepted"`
	Mode      CapacityMode     `json:"mode"`
}

type Allocator struct {
	ledger  *Ledger
	builder ViewBuilder
}

func NewAllocator(ledger *Ledger) Allocator {
	return Allocator{ledger: ledger, builder: NewViewBuilder(ledger)}
}

func (a Allocator) BuildBatch(options PlannerOptions) AllocationBatch {
	batch := AllocationBatch{
		ID:    a.ledger.Seq.Next("allocation"),
		Clock: a.ledger.Clock,
		Plans: []AllocationPlan{},
		Mode:  CapacityObserved,
	}
	ranked := a.builder.RankedRoutes(options)
	for _, candidate := range ranked {
		route := a.ledger.Routes.MustGet(candidate.RouteID)
		amount := candidate.Capacity.Assignable.Min(route.RemainingDemand())
		if options.MinimumBatch > 0 && amount < options.MinimumBatch {
			continue
		}
		if amount < route.MinBatch {
			continue
		}
		projected := Amount(0)
		if candidate.Capacity.UsesProjection() && amount > candidate.Capacity.Free {
			projected = amount - candidate.Capacity.Free
			batch.Mode = CapacityProjected
		}
		plan := AllocationPlan{
			ID:             a.ledger.Seq.Next("plan"),
			Route:          route.ID,
			SourceVault:    route.SourceVault,
			TargetVault:    route.TargetVault,
			Asset:          route.Asset,
			Amount:         amount,
			Mode:           candidate.Capacity.Mode,
			ProjectedShare: projected,
			Reason:         "demand-weighted allocation",
		}
		batch.Requested += route.RemainingDemand()
		batch.Accepted += amount
		batch.Plans = append(batch.Plans, plan)
	}
	return batch
}

func (a Allocator) Execute(batch AllocationBatch, kind ReservationKind) ([]string, []error) {
	ids := []string{}
	errors := []error{}
	for _, plan := range batch.Plans {
		allowProjection := plan.Mode == CapacityProjected && plan.ProjectedShare > 0
		id, err := a.ledger.OpenSettlement(plan.Route, plan.Amount, kind, allowProjection)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		ids = append(ids, id)
		a.ledger.Journal.Record(a.ledger.Clock, "allocation.executed", plan.ID, plan.SourceVault, plan.Route, plan.Asset, plan.Amount, plan.Reason, map[string]string{
			"settlement": id,
			"mode":       string(plan.Mode),
		})
	}
	return ids, errors
}

func (a Allocator) ExecuteObserved(minimum Amount) ([]string, []error) {
	batch := a.BuildBatch(PlannerOptions{UseProjected: false, MinimumBatch: minimum})
	return a.Execute(batch, ReservationAllocation)
}
