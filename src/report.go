package main

type TotalsReport struct {
	Reserves    []AmountBucket `json:"reserves"`
	Liabilities []AmountBucket `json:"liabilities"`
	PendingIn   []AmountBucket `json:"pending_in"`
	PendingOut  []AmountBucket `json:"pending_out"`
	Fees        []AmountBucket `json:"fees"`
}

type Report struct {
	Lab          string               `json:"lab"`
	Scenario     string               `json:"scenario"`
	NetworkID    string               `json:"network_id"`
	Clock        int64                `json:"clock"`
	StateDigest  string               `json:"state_digest"`
	Assets       []Asset              `json:"assets"`
	Vaults       []Vault              `json:"vaults"`
	Routes       []Route              `json:"routes"`
	Accounts     []Account            `json:"accounts"`
	Settlements  []Settlement         `json:"settlements"`
	Reservations []Reservation        `json:"reservations"`
	Forecasts    []Forecast           `json:"forecasts"`
	Views        []LiquidityView      `json:"views"`
	Rebalances   []RebalanceResult    `json:"rebalances"`
	Liquidations []LiquidationReceipt `json:"liquidations"`
	Totals       TotalsReport         `json:"totals"`
	Risk         RiskMetrics          `json:"risk"`
	Invariants   InvariantReport      `json:"invariants"`
	Events       []Event              `json:"events"`
	Notes        []string             `json:"notes"`
}

func BuildReport(run *ScenarioRun) Report {
	ledger := run.Ledger
	risk := NewRiskEngine(ledger)
	notes := append([]string{}, ledger.Notes...)
	notes = append(notes, run.ScenarioNotes...)
	return Report{
		Lab:          ledger.Lab,
		Scenario:     run.Name,
		NetworkID:    ledger.NetworkID,
		Clock:        ledger.Clock,
		StateDigest:  ledger.Digest(),
		Assets:       ledger.Assets.List(),
		Vaults:       ledger.Vaults.List(),
		Routes:       ledger.Routes.List(),
		Accounts:     ledger.Accounts.List(),
		Settlements:  ledger.Settlements.List(),
		Reservations: ledger.Reservations.List(),
		Forecasts:    ledger.Forecasts.List(),
		Views:        run.Views,
		Rebalances:   run.Rebalances,
		Liquidations: run.Liquidations,
		Totals:       buildTotals(ledger),
		Risk:         risk.Metrics(),
		Invariants:   risk.Invariants(),
		Events:       ledger.Journal.List(),
		Notes:        notes,
	}
}

func buildTotals(ledger *Ledger) TotalsReport {
	reserves := []AmountBucket{}
	liabilities := []AmountBucket{}
	pendingIn := []AmountBucket{}
	pendingOut := []AmountBucket{}
	fees := []AmountBucket{}
	for _, asset := range ledger.Assets.List() {
		reserveTotal := Amount(0)
		liabilityTotal := Amount(0)
		pendingInTotal := Amount(0)
		pendingOutTotal := Amount(0)
		for _, vault := range ledger.Vaults.List() {
			if vault.Asset != asset.ID {
				continue
			}
			reserveTotal += vault.Reserve
			liabilityTotal += vault.Liability
			pendingInTotal += vault.PendingIn
			pendingOutTotal += vault.PendingOut
		}
		feeTotal := Amount(0)
		for _, settlement := range ledger.Settlements.List() {
			if settlement.Asset == asset.ID && settlement.Status == SettlementFinalized {
				feeTotal += settlement.Fee
			}
		}
		reserves = append(reserves, Bucket(asset.ID, reserveTotal))
		liabilities = append(liabilities, Bucket(asset.ID, liabilityTotal))
		pendingIn = append(pendingIn, Bucket(asset.ID, pendingInTotal))
		pendingOut = append(pendingOut, Bucket(asset.ID, pendingOutTotal))
		fees = append(fees, Bucket(asset.ID, feeTotal))
	}
	return TotalsReport{
		Reserves:    reserves,
		Liabilities: liabilities,
		PendingIn:   pendingIn,
		PendingOut:  pendingOut,
		Fees:        fees,
	}
}

func ValidateReport(report Report) bool {
	return report.Invariants.VaultsNonNegative &&
		report.Invariants.AccountsNonNegative &&
		report.Invariants.SettlementLinksValid &&
		report.Invariants.ForecastLinksValid &&
		report.Invariants.RoutesWithinDemand &&
		report.Invariants.CapacityViewsNonNegative &&
		report.Invariants.WithdrawalsOpen
}
