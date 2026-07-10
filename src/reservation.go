package main

import "fmt"

type ReservationStatus string

const (
	ReservationHeld     ReservationStatus = "held"
	ReservationConsumed ReservationStatus = "consumed"
	ReservationReleased ReservationStatus = "released"
	ReservationExpired  ReservationStatus = "expired"
)

type ReservationKind string

const (
	ReservationAllocation  ReservationKind = "allocation"
	ReservationRebalance   ReservationKind = "rebalance"
	ReservationLiquidation ReservationKind = "liquidation"
)

type Reservation struct {
	ID           string            `json:"id"`
	Vault        string            `json:"vault"`
	Route        string            `json:"route"`
	Asset        string            `json:"asset"`
	Amount       Amount            `json:"amount"`
	Kind         ReservationKind   `json:"kind"`
	CreatedAt    int64             `json:"created_at"`
	ExpiresAt    int64             `json:"expires_at"`
	Status       ReservationStatus `json:"status"`
	SettlementID string            `json:"settlement_id,omitempty"`
	Note         string            `json:"note,omitempty"`
}

func NewReservation(id string, vault string, route string, asset string, amount Amount, kind ReservationKind, clock int64, ttl int64) Reservation {
	return Reservation{
		ID:        id,
		Vault:     vault,
		Route:     route,
		Asset:     asset,
		Amount:    amount,
		Kind:      kind,
		CreatedAt: clock,
		ExpiresAt: clock + ttl,
		Status:    ReservationHeld,
	}
}

func (r Reservation) Active(clock int64) bool {
	return r.Status == ReservationHeld && clock <= r.ExpiresAt
}

func (r Reservation) Expired(clock int64) bool {
	return r.Status == ReservationHeld && clock > r.ExpiresAt
}

func (r *Reservation) AttachSettlement(id string) {
	r.SettlementID = id
}

func (r *Reservation) Consume() {
	r.Status = ReservationConsumed
}

func (r *Reservation) Release(note string) {
	r.Status = ReservationReleased
	r.Note = note
}

func (r *Reservation) Expire() {
	r.Status = ReservationExpired
}

type ReservationBook struct {
	reservations map[string]*Reservation
	order        []string
}

func NewReservationBook() *ReservationBook {
	return &ReservationBook{reservations: map[string]*Reservation{}, order: []string{}}
}

func (b *ReservationBook) Add(reservation Reservation) error {
	if reservation.ID == "" {
		return fmt.Errorf("reservation id missing")
	}
	if reservation.Amount <= 0 {
		return fmt.Errorf("reservation %s amount must be positive", reservation.ID)
	}
	if _, ok := b.reservations[reservation.ID]; !ok {
		b.order = append(b.order, reservation.ID)
	}
	copy := reservation
	b.reservations[reservation.ID] = &copy
	return nil
}

func (b *ReservationBook) Get(id string) (*Reservation, bool) {
	reservation, ok := b.reservations[id]
	return reservation, ok
}

func (b *ReservationBook) MustGet(id string) *Reservation {
	reservation, ok := b.Get(id)
	if !ok {
		panic("missing reservation " + id)
	}
	return reservation
}

func (b *ReservationBook) List() []Reservation {
	out := make([]Reservation, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, *b.reservations[id])
	}
	return out
}

func (b *ReservationBook) ActiveForVault(vaultID string, clock int64) []Reservation {
	out := []Reservation{}
	for _, id := range b.order {
		reservation := b.reservations[id]
		if reservation.Vault == vaultID && reservation.Active(clock) {
			out = append(out, *reservation)
		}
	}
	return out
}

func (b *ReservationBook) Expire(clock int64) []string {
	expired := []string{}
	for _, id := range b.order {
		reservation := b.reservations[id]
		if reservation.Expired(clock) {
			reservation.Expire()
			expired = append(expired, id)
		}
	}
	return expired
}

func (b *ReservationBook) HeldTotal(vaultID string) Amount {
	total := Amount(0)
	for _, reservation := range b.reservations {
		if reservation.Vault == vaultID && reservation.Status == ReservationHeld {
			total += reservation.Amount
		}
	}
	return total
}
