package main

import "fmt"

type Ledger struct {
	Lab          string
	NetworkID    string
	Clock        int64
	Assets       *AssetRegistry
	Accounts     *AccountBook
	Vaults       *VaultBook
	Routes       *RouteBook
	Settlements  *SettlementBook
	Reservations *ReservationBook
	Forecasts    *ForecastBook
	Journal      *Journal
	Seq          *Sequencer
	Policy       ProtocolPolicy
	Notes        []string
}

func NewLedger() *Ledger {
	policy := DefaultPolicy()
	return &Ledger{
		Lab:          "QuarryDTL",
		NetworkID:    "quarry-local-liquidity",
		Clock:        1_700_000_000,
		Assets:       NewAssetRegistry(),
		Accounts:     NewAccountBook(),
		Vaults:       NewVaultBook(),
		Routes:       NewRouteBook(),
		Settlements:  NewSettlementBook(),
		Reservations: NewReservationBook(),
		Forecasts:    NewForecastBook(),
		Journal:      NewJournal(),
		Seq:          NewSequencer("qdtl"),
		Policy:       policy,
		Notes:        []string{},
	}
}

func (l *Ledger) AddNote(note string) {
	if note != "" {
		l.Notes = append(l.Notes, note)
	}
}

func (l *Ledger) AddAsset(asset Asset) {
	l.Assets.MustAdd(asset)
	l.Journal.Record(l.Clock, "asset.registered", asset.ID, "", "", asset.ID, 0, "asset registered", nil)
}

func (l *Ledger) AddAccount(account *Account) {
	l.Accounts.MustAdd(account)
	l.Journal.Record(l.Clock, "account.registered", account.ID, "", "", "", 0, "account registered", nil)
}

func (l *Ledger) AddVault(vault *Vault) {
	vault.MinBuffer = l.Policy.MinBuffer(vault)
	l.Vaults.MustAdd(vault)
	l.Journal.Record(l.Clock, "vault.registered", vault.ID, vault.ID, "", vault.Asset, vault.Reserve, "vault registered", nil)
}

func (l *Ledger) AddRoute(route *Route) {
	l.Routes.MustAdd(route)
	l.Journal.Record(l.Clock, "route.registered", route.ID, "", route.ID, route.Asset, route.Demand, "route registered", nil)
}

func (l *Ledger) Advance(ticks int64) {
	if ticks <= 0 {
		return
	}
	for i := int64(0); i < ticks; i++ {
		l.Clock++
		l.Forecasts.Observe(l.Clock)
		expired := l.Reservations.Expire(l.Clock)
		for _, id := range expired {
			l.Journal.Record(l.Clock, "reservation.expired", id, "", "", "", 0, "reservation expired", nil)
		}
	}
}

func (l *Ledger) setRouteLatency(route *Route) {
	latency := l.Policy.SettlementLatency(route)
	if latency != route.ExpectedLatency {
		route.ExpectedLatency = latency
	}
}

func (l *Ledger) ReserveVault(vault *Vault, amount Amount, reason string, allowProjection bool) (Amount, error) {
	if amount <= 0 {
		return 0, fmt.Errorf("reserve amount must be positive")
	}
	available := vault.Available()
	if available >= amount {
		return 0, vault.ReserveFor(reason, amount)
	}
	if !allowProjection || !l.Policy.ProjectedCapacityEnabled {
		return 0, fmt.Errorf("vault %s has insufficient assignable liquidity", vault.ID)
	}
	if available > 0 {
		vault.Idle -= available
	}
	vault.Reserved += amount
	return amount - available, nil
}

func (l *Ledger) OpenSettlement(routeID string, amount Amount, kind ReservationKind, allowProjection bool) (string, error) {
	route, ok := l.Routes.Get(routeID)
	if !ok {
		return "", fmt.Errorf("route %s not found", routeID)
	}
	if !route.IsOpen() {
		return "", fmt.Errorf("route %s is not open", routeID)
	}
	if amount < route.MinBatch {
		return "", fmt.Errorf("amount below route min batch")
	}
	if route.RemainingDemand() < amount {
		return "", fmt.Errorf("amount exceeds remaining route demand")
	}
	source := l.Vaults.MustGet(route.SourceVault)
	target := l.Vaults.MustGet(route.TargetVault)
	if source.Asset != route.Asset || target.Asset != route.Asset {
		return "", fmt.Errorf("route %s asset mismatch", route.ID)
	}
	l.setRouteLatency(route)
	projected, err := l.ReserveVault(source, amount, string(kind), allowProjection)
	if err != nil {
		return "", err
	}
	settlementID := l.Seq.Next("settlement")
	reservationID := l.Seq.Next("reservation")
	forecastID := l.Seq.Next("forecast")
	reservation := NewReservation(reservationID, source.ID, route.ID, route.Asset, amount, kind, l.Clock, l.Policy.ReservationTTL)
	reservation.AttachSettlement(settlementID)
	if err := l.Reservations.Add(reservation); err != nil {
		return "", err
	}
	settlement := NewSettlement(settlementID, route, amount, l.Clock)
	settlement.Projected = projected
	settlement.Prepare(reservationID, forecastID)
	settlement.Submit()
	if err := source.OpenOutflow(amount); err != nil {
		return "", err
	}
	res := l.Reservations.MustGet(reservationID)
	res.Consume()
	net := amount - settlement.Fee
	if err := target.ExpectInflow(net); err != nil {
		return "", err
	}
	forecast := NewForecast(forecastID, settlement)
	forecast.Amount = net
	if err := l.Settlements.Add(settlement); err != nil {
		return "", err
	}
	if err := l.Forecasts.Add(forecast); err != nil {
		return "", err
	}
	route.RegisterAllocation(amount)
	l.Journal.Record(l.Clock, "settlement.submitted", settlementID, source.ID, route.ID, route.Asset, amount, "settlement submitted", map[string]string{
		"target":      target.ID,
		"reservation": reservationID,
		"forecast":    forecastID,
	})
	return settlementID, nil
}

func (l *Ledger) CompleteSettlement(settlementID string) error {
	settlement := l.Settlements.MustGet(settlementID)
	if !settlement.Active() {
		return fmt.Errorf("settlement %s is not active", settlementID)
	}
	source := l.Vaults.MustGet(settlement.SourceVault)
	target := l.Vaults.MustGet(settlement.TargetVault)
	route := l.Routes.MustGet(settlement.Route)
	net := settlement.Amount - settlement.Fee
	if settlement.Projected > 0 {
		drain := settlement.Projected.Min(source.Idle)
		source.Idle -= drain
	}
	if err := source.CompleteOutflow(settlement.Amount); err != nil {
		return err
	}
	if err := target.CompleteInflow(net); err != nil {
		return err
	}
	settlement.Finalize()
	l.Forecasts.SettleBySource(settlement.ID)
	route.RegisterSettlement(settlement.Amount)
	l.Journal.Record(l.Clock, "settlement.finalized", settlement.ID, target.ID, route.ID, settlement.Asset, settlement.Amount, "settlement finalized", map[string]string{
		"source": source.ID,
	})
	return nil
}

func (l *Ledger) FailSettlement(settlementID string, reason string) error {
	settlement := l.Settlements.MustGet(settlementID)
	if !settlement.Active() {
		return fmt.Errorf("settlement %s is not active", settlementID)
	}
	source := l.Vaults.MustGet(settlement.SourceVault)
	target := l.Vaults.MustGet(settlement.TargetVault)
	route := l.Routes.MustGet(settlement.Route)
	net := settlement.Amount - settlement.Fee
	if err := source.RevertOutflow(settlement.Amount); err != nil {
		return err
	}
	if err := target.CancelInflow(net); err != nil {
		return err
	}
	settlement.Fail(reason)
	l.Forecasts.VoidBySource(settlement.ID)
	route.RegisterFailure(settlement.Amount)
	l.Journal.Record(l.Clock, "settlement.failed", settlement.ID, source.ID, route.ID, settlement.Asset, settlement.Amount, reason, map[string]string{
		"target": target.ID,
	})
	return nil
}

func (l *Ledger) CompleteDueSettlements() []string {
	completed := []string{}
	for _, id := range l.Settlements.Due(l.Clock) {
		if err := l.CompleteSettlement(id); err == nil {
			completed = append(completed, id)
		}
	}
	return completed
}

func (l *Ledger) FailDueSettlements(reason string, max int) []string {
	failed := []string{}
	for _, id := range l.Settlements.Due(l.Clock) {
		if max > 0 && len(failed) >= max {
			break
		}
		if err := l.FailSettlement(id, reason); err == nil {
			failed = append(failed, id)
		}
	}
	return failed
}

func (l *Ledger) TotalReserve(asset string) Amount {
	return l.Vaults.TotalReserve(asset)
}

func (l *Ledger) TotalLiability(asset string) Amount {
	return l.Vaults.TotalLiability(asset)
}

func (l *Ledger) Digest() string {
	return DigestJSON(map[string]any{
		"clock":       l.Clock,
		"assets":      l.Assets.List(),
		"vaults":      l.Vaults.List(),
		"routes":      l.Routes.List(),
		"settlements": l.Settlements.List(),
		"forecasts":   l.Forecasts.List(),
		"events":      l.Journal.List(),
	})
}
