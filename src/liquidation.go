package main

type LiquidationCandidate struct {
	Vault       string `json:"vault"`
	Asset       string `json:"asset"`
	CoverageBps int64  `json:"coverage_bps"`
	Shortfall   Amount `json:"shortfall"`
	Penalty     Amount `json:"penalty"`
}

type LiquidationReceipt struct {
	ID         string                 `json:"id"`
	Clock      int64                  `json:"clock"`
	Candidates []LiquidationCandidate `json:"candidates"`
	Recovered  Amount                 `json:"recovered"`
	Penalties  Amount                 `json:"penalties"`
}

type LiquidationEngine struct {
	ledger *Ledger
}

func NewLiquidationEngine(ledger *Ledger) LiquidationEngine {
	return LiquidationEngine{ledger: ledger}
}

func (e LiquidationEngine) Candidates() []LiquidationCandidate {
	out := []LiquidationCandidate{}
	for _, snapshot := range e.ledger.Vaults.List() {
		vault := e.ledger.Vaults.MustGet(snapshot.ID)
		if !e.ledger.Policy.Liquidatable(vault) {
			continue
		}
		required := vault.Liability.MulBps(e.ledger.Policy.LiquidationThresholdBps)
		if required <= vault.Reserve {
			continue
		}
		shortfall := required - vault.Reserve
		out = append(out, LiquidationCandidate{
			Vault:       vault.ID,
			Asset:       vault.Asset,
			CoverageBps: vault.CoverageBps(),
			Shortfall:   shortfall,
			Penalty:     e.ledger.Policy.RecoveryPenalty(shortfall),
		})
	}
	return out
}

func (e LiquidationEngine) Execute() LiquidationReceipt {
	receipt := LiquidationReceipt{
		ID:         e.ledger.Seq.Next("liquidation"),
		Clock:      e.ledger.Clock,
		Candidates: e.Candidates(),
	}
	for _, candidate := range receipt.Candidates {
		vault := e.ledger.Vaults.MustGet(candidate.Vault)
		recoverable := candidate.Shortfall.Min(vault.Idle)
		if recoverable <= 0 {
			continue
		}
		penalty := candidate.Penalty.Min(recoverable)
		_ = vault.ApplyPenalty(penalty)
		_ = vault.BurnLiability(recoverable - penalty)
		receipt.Recovered += recoverable
		receipt.Penalties += penalty
		e.ledger.Journal.Record(e.ledger.Clock, "liquidation.recovered", receipt.ID, vault.ID, "", vault.Asset, recoverable, "vault recovery applied", map[string]string{
			"coverage_bps": stringInt(candidate.CoverageBps),
		})
	}
	e.ledger.Journal.Record(e.ledger.Clock, "liquidation.executed", receipt.ID, "", "", "", receipt.Recovered, "liquidation cycle executed", nil)
	return receipt
}

func stringInt(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	buf := [20]byte{}
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
