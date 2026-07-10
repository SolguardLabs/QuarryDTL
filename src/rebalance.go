package main

type RebalanceIntent struct {
	ID             string `json:"id"`
	RequestedBy    string `json:"requested_by"`
	Reason         string `json:"reason"`
	UseProjection  bool   `json:"use_projection"`
	MaxSettlements int    `json:"max_settlements"`
	MinimumBatch   Amount `json:"minimum_batch"`
	CreatedAt      int64  `json:"created_at"`
}

type RebalanceResult struct {
	IntentID       string           `json:"intent_id"`
	BatchID        string           `json:"batch_id"`
	Submitted      []string         `json:"submitted"`
	Rejected       int              `json:"rejected"`
	AcceptedAmount Amount           `json:"accepted_amount"`
	Requested      Amount           `json:"requested"`
	Mode           CapacityMode     `json:"mode"`
	Plans          []AllocationPlan `json:"plans"`
}

type RebalanceEngine struct {
	ledger    *Ledger
	allocator Allocator
}

func NewRebalanceEngine(ledger *Ledger) RebalanceEngine {
	return RebalanceEngine{ledger: ledger, allocator: NewAllocator(ledger)}
}

func (r RebalanceEngine) NewIntent(requestedBy string, reason string, useProjection bool) RebalanceIntent {
	return RebalanceIntent{
		ID:             r.ledger.Seq.Next("rebalance"),
		RequestedBy:    requestedBy,
		Reason:         reason,
		UseProjection:  useProjection,
		MaxSettlements: 8,
		MinimumBatch:   50_000,
		CreatedAt:      r.ledger.Clock,
	}
}

func (r RebalanceEngine) Plan(intent RebalanceIntent) AllocationBatch {
	batch := r.allocator.BuildBatch(PlannerOptions{
		UseProjected: intent.UseProjection,
		MinimumBatch: intent.MinimumBatch,
	})
	if intent.MaxSettlements > 0 && len(batch.Plans) > intent.MaxSettlements {
		batch.Plans = batch.Plans[:intent.MaxSettlements]
		accepted := Amount(0)
		mode := CapacityObserved
		for _, plan := range batch.Plans {
			accepted += plan.Amount
			if plan.Mode == CapacityProjected {
				mode = CapacityProjected
			}
		}
		batch.Accepted = accepted
		batch.Mode = mode
	}
	return batch
}

func (r RebalanceEngine) Execute(intent RebalanceIntent) RebalanceResult {
	batch := r.Plan(intent)
	submitted, errors := r.allocator.Execute(batch, ReservationRebalance)
	result := RebalanceResult{
		IntentID:       intent.ID,
		BatchID:        batch.ID,
		Submitted:      submitted,
		Rejected:       len(errors),
		AcceptedAmount: batch.Accepted,
		Requested:      batch.Requested,
		Mode:           batch.Mode,
		Plans:          batch.Plans,
	}
	r.ledger.Journal.Record(r.ledger.Clock, "rebalance.executed", intent.ID, "", "", "", batch.Accepted, intent.Reason, map[string]string{
		"batch": batch.ID,
		"mode":  string(batch.Mode),
	})
	return result
}

func (r RebalanceEngine) ExecuteCycle(requestedBy string, reason string, useProjection bool) RebalanceResult {
	intent := r.NewIntent(requestedBy, reason, useProjection)
	return r.Execute(intent)
}

func (r RebalanceEngine) Preview(useProjection bool) LiquidityView {
	return NewViewBuilder(r.ledger).Build(PlannerOptions{UseProjected: useProjection, MinimumBatch: 0})
}
