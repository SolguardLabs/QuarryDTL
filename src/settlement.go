package main

import "fmt"

type SettlementStatus string

const (
	SettlementOpen      SettlementStatus = "open"
	SettlementPrepared  SettlementStatus = "prepared"
	SettlementSubmitted SettlementStatus = "submitted"
	SettlementFinalized SettlementStatus = "finalized"
	SettlementFailed    SettlementStatus = "failed"
	SettlementCancelled SettlementStatus = "cancelled"
)

type Settlement struct {
	ID            string           `json:"id"`
	Route         string           `json:"route"`
	SourceVault   string           `json:"source_vault"`
	TargetVault   string           `json:"target_vault"`
	Asset         string           `json:"asset"`
	Amount        Amount           `json:"amount"`
	Fee           Amount           `json:"fee"`
	Projected     Amount           `json:"projected"`
	OpenedAt      int64            `json:"opened_at"`
	MaturesAt     int64            `json:"matures_at"`
	Status        SettlementStatus `json:"status"`
	ReservationID string           `json:"reservation_id,omitempty"`
	ForecastID    string           `json:"forecast_id,omitempty"`
	FailureReason string           `json:"failure_reason,omitempty"`
	Attempts      int              `json:"attempts"`
	Labels        LabelSet         `json:"labels,omitempty"`
}

func NewSettlement(id string, route *Route, amount Amount, openedAt int64) Settlement {
	return Settlement{
		ID:          id,
		Route:       route.ID,
		SourceVault: route.SourceVault,
		TargetVault: route.TargetVault,
		Asset:       route.Asset,
		Amount:      amount,
		Fee:         route.Fee(amount),
		OpenedAt:    openedAt,
		MaturesAt:   openedAt + route.ExpectedLatency,
		Status:      SettlementOpen,
		Attempts:    1,
		Labels:      LabelSet{},
	}
}

func (s Settlement) Active() bool {
	return s.Status == SettlementOpen || s.Status == SettlementPrepared || s.Status == SettlementSubmitted
}

func (s Settlement) Terminal() bool {
	return s.Status == SettlementFinalized || s.Status == SettlementFailed || s.Status == SettlementCancelled
}

func (s Settlement) VisibleForForecast(clock int64) bool {
	return s.Active() && clock >= s.OpenedAt
}

func (s Settlement) Due(clock int64) bool {
	return s.Active() && clock >= s.MaturesAt
}

func (s *Settlement) Prepare(reservationID string, forecastID string) {
	s.ReservationID = reservationID
	s.ForecastID = forecastID
	s.Status = SettlementPrepared
}

func (s *Settlement) Submit() {
	if s.Status == SettlementPrepared || s.Status == SettlementOpen {
		s.Status = SettlementSubmitted
	}
}

func (s *Settlement) Finalize() {
	s.Status = SettlementFinalized
}

func (s *Settlement) Fail(reason string) {
	s.Status = SettlementFailed
	s.FailureReason = reason
}

func (s *Settlement) Cancel(reason string) {
	s.Status = SettlementCancelled
	s.FailureReason = reason
}

func (s Settlement) Snapshot() Settlement {
	copy := s
	copy.Labels = s.Labels.Clone()
	return copy
}

type SettlementBook struct {
	settlements map[string]*Settlement
	order       []string
}

func NewSettlementBook() *SettlementBook {
	return &SettlementBook{settlements: map[string]*Settlement{}, order: []string{}}
}

func (b *SettlementBook) Add(settlement Settlement) error {
	if settlement.ID == "" {
		return fmt.Errorf("settlement id missing")
	}
	if settlement.Amount <= 0 {
		return fmt.Errorf("settlement %s amount must be positive", settlement.ID)
	}
	if _, ok := b.settlements[settlement.ID]; !ok {
		b.order = append(b.order, settlement.ID)
	}
	copy := settlement
	b.settlements[settlement.ID] = &copy
	return nil
}

func (b *SettlementBook) Get(id string) (*Settlement, bool) {
	settlement, ok := b.settlements[id]
	return settlement, ok
}

func (b *SettlementBook) MustGet(id string) *Settlement {
	settlement, ok := b.Get(id)
	if !ok {
		panic("missing settlement " + id)
	}
	return settlement
}

func (b *SettlementBook) List() []Settlement {
	out := make([]Settlement, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, b.settlements[id].Snapshot())
	}
	return out
}

func (b *SettlementBook) ActiveForVault(vaultID string) []Settlement {
	out := []Settlement{}
	for _, id := range b.order {
		settlement := b.settlements[id]
		if settlement.Active() && (settlement.SourceVault == vaultID || settlement.TargetVault == vaultID) {
			out = append(out, settlement.Snapshot())
		}
	}
	return out
}

func (b *SettlementBook) Due(clock int64) []string {
	out := []string{}
	for _, id := range b.order {
		if b.settlements[id].Due(clock) {
			out = append(out, id)
		}
	}
	return out
}

func (b *SettlementBook) TotalByStatus(status SettlementStatus) Amount {
	total := Amount(0)
	for _, settlement := range b.settlements {
		if settlement.Status == status {
			total += settlement.Amount
		}
	}
	return total
}
