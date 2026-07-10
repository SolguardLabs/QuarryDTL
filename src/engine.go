package main

import "fmt"

type ScenarioRun struct {
	Name          string               `json:"name"`
	Ledger        *Ledger              `json:"-"`
	Views         []LiquidityView      `json:"views"`
	Rebalances    []RebalanceResult    `json:"rebalances"`
	Liquidations  []LiquidationReceipt `json:"liquidations"`
	Submitted     []string             `json:"submitted"`
	Completed     []string             `json:"completed"`
	Failed        []string             `json:"failed"`
	ScenarioNotes []string             `json:"scenario_notes"`
}

func AvailableScenarios() []string {
	return []string{
		"baseline",
		"allocation",
		"rebalance",
		"liquidation",
		"capacity-cycle",
		"operator-cycle",
	}
}

func RunScenario(name string) (*ScenarioRun, error) {
	switch name {
	case "baseline":
		return scenarioBaseline(), nil
	case "allocation":
		return scenarioAllocation(), nil
	case "rebalance":
		return scenarioRebalance(), nil
	case "liquidation":
		return scenarioLiquidation(), nil
	case "capacity-cycle":
		return scenarioCapacityCycle(), nil
	case "operator-cycle":
		return scenarioOperatorCycle(), nil
	default:
		return nil, fmt.Errorf("unknown scenario %q", name)
	}
}

func newScenarioRun(name string, ledger *Ledger) *ScenarioRun {
	return &ScenarioRun{
		Name:          name,
		Ledger:        ledger,
		Views:         []LiquidityView{},
		Rebalances:    []RebalanceResult{},
		Liquidations:  []LiquidationReceipt{},
		Submitted:     []string{},
		Completed:     []string{},
		Failed:        []string{},
		ScenarioNotes: []string{},
	}
}

func (s *ScenarioRun) AddView(view LiquidityView) {
	s.Views = append(s.Views, view)
}

func (s *ScenarioRun) AddRebalance(result RebalanceResult) {
	s.Rebalances = append(s.Rebalances, result)
	s.Submitted = append(s.Submitted, result.Submitted...)
}

func (s *ScenarioRun) AddLiquidation(receipt LiquidationReceipt) {
	s.Liquidations = append(s.Liquidations, receipt)
}

func (s *ScenarioRun) AddCompleted(ids []string) {
	s.Completed = append(s.Completed, ids...)
}

func (s *ScenarioRun) AddFailed(ids []string) {
	s.Failed = append(s.Failed, ids...)
}

func (s *ScenarioRun) Note(value string) {
	if value != "" {
		s.ScenarioNotes = append(s.ScenarioNotes, value)
		s.Ledger.AddNote(value)
	}
}

func BuildSeedLedger() *Ledger {
	ledger := NewLedger()
	usdc := NewAsset("usdc", "USDC", 6, AssetStable)
	usdc.SettlementBps = 4
	usdt := NewAsset("usdt", "USDT", 6, AssetStable)
	usdt.SettlementBps = 3
	qeth := NewAsset("qeth", "qETH", 8, AssetCollateral)
	qeth.RiskWeightBps = 8_800
	qeth.LiquidationBps = 125
	ledger.AddAsset(usdc)
	ledger.AddAsset(usdt)
	ledger.AddAsset(qeth)

	protocol := NewAccount("protocol", "protocol")
	treasury := NewAccount("treasury", "treasury")
	operator := NewAccount("operator-a", "operator")
	marketMaker := NewAccount("market-maker", "allocator")
	_ = protocol.Deposit("usdc", 25_000)
	_ = treasury.Deposit("usdc", 500_000)
	_ = operator.Deposit("usdc", 100_000)
	_ = marketMaker.Deposit("usdt", 250_000)
	ledger.AddAccount(protocol)
	ledger.AddAccount(treasury)
	ledger.AddAccount(operator)
	ledger.AddAccount(marketMaker)

	alpha := NewVault("vault-alpha-usdc", "usdc", 4_000_000)
	alpha.Priority = 5
	alpha.Liability = 1_750_000
	alpha.Strategy = "source-buffer"
	beta := NewVault("vault-beta-usdc", "usdc", 160_000)
	beta.Priority = 8
	beta.Liability = 120_000
	beta.Strategy = "demand-edge"
	delta := NewVault("vault-delta-usdc", "usdc", 80_000)
	delta.Priority = 4
	delta.Liability = 40_000
	delta.Strategy = "merchant-payout"
	gamma := NewVault("vault-gamma-usdc", "usdc", 650_000)
	gamma.Priority = 2
	gamma.Liability = 900_000
	gamma.Strategy = "slow-withdrawal"
	south := NewVault("vault-south-usdt", "usdt", 1_250_000)
	south.Priority = 6
	south.Liability = 730_000
	north := NewVault("vault-north-usdt", "usdt", 260_000)
	north.Priority = 7
	north.Liability = 150_000
	eth := NewVault("vault-eth-collateral", "qeth", 75_000_000)
	eth.Priority = 1
	eth.Liability = 22_000_000
	ledger.AddVault(alpha)
	ledger.AddVault(beta)
	ledger.AddVault(delta)
	ledger.AddVault(gamma)
	ledger.AddVault(south)
	ledger.AddVault(north)
	ledger.AddVault(eth)

	alphaBeta := NewRoute("route-alpha-beta", "vault-alpha-usdc", "vault-beta-usdc", "usdc", 700_000)
	alphaBeta.Priority = 7
	alphaBeta.MinBatch = 100_000
	alphaBeta.ExpectedLatency = 2
	alphaBeta.Tags.Set("market", "core")
	betaDelta := NewRoute("route-beta-delta", "vault-beta-usdc", "vault-delta-usdc", "usdc", 650_000)
	betaDelta.Priority = 9
	betaDelta.MinBatch = 250_000
	betaDelta.ExpectedLatency = 2
	betaDelta.Tags.Set("market", "merchant")
	southNorth := NewRoute("route-south-north", "vault-south-usdt", "vault-north-usdt", "usdt", 460_000)
	southNorth.Priority = 6
	southNorth.MinBatch = 75_000
	southNorth.ExpectedLatency = 3
	gammaAlpha := NewRoute("route-gamma-alpha", "vault-gamma-usdc", "vault-alpha-usdc", "usdc", 180_000)
	gammaAlpha.Priority = 2
	gammaAlpha.MinBatch = 50_000
	gammaAlpha.ExpectedLatency = 4
	ledger.AddRoute(alphaBeta)
	ledger.AddRoute(betaDelta)
	ledger.AddRoute(southNorth)
	ledger.AddRoute(gammaAlpha)
	ledger.AddNote("seeded quarry vault mesh with deterministic reserves and demand routes")
	return ledger
}

func scenarioBaseline() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := newScenarioRun("baseline", ledger)
	builder := NewViewBuilder(ledger)
	run.AddView(builder.Build(PlannerOptions{UseProjected: false}))
	run.Note("baseline report without settlement execution")
	return run
}

func scenarioAllocation() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := newScenarioRun("allocation", ledger)
	allocator := NewAllocator(ledger)
	run.AddView(NewViewBuilder(ledger).Build(PlannerOptions{UseProjected: false}))
	batch := allocator.BuildBatch(PlannerOptions{UseProjected: false, MinimumBatch: 75_000})
	submitted, _ := allocator.Execute(batch, ReservationAllocation)
	run.Submitted = append(run.Submitted, submitted...)
	ledger.Advance(4)
	run.AddCompleted(ledger.CompleteDueSettlements())
	run.AddView(NewViewBuilder(ledger).Build(PlannerOptions{UseProjected: false}))
	run.Note("observed idle liquidity assigned to open demand routes")
	return run
}

func scenarioRebalance() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := newScenarioRun("rebalance", ledger)
	first, err := ledger.OpenSettlement("route-alpha-beta", 700_000, ReservationAllocation, false)
	if err == nil {
		run.Submitted = append(run.Submitted, first)
	}
	ledger.Advance(1)
	engine := NewRebalanceEngine(ledger)
	run.AddView(engine.Preview(true))
	result := engine.ExecuteCycle("operator-a", "edge demand rebalance", true)
	run.AddRebalance(result)
	ledger.Advance(1)
	run.AddCompleted(ledger.CompleteDueSettlements())
	ledger.Advance(1)
	run.AddCompleted(ledger.CompleteDueSettlements())
	run.AddView(engine.Preview(true))
	run.Note("rebalance used forecast-aware capacity and settled in route order")
	return run
}

func scenarioLiquidation() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := newScenarioRun("liquidation", ledger)
	liquidator := NewLiquidationEngine(ledger)
	before := liquidator.Candidates()
	if len(before) > 0 {
		run.Note("liquidation candidates detected by coverage policy")
	}
	receipt := liquidator.Execute()
	run.AddLiquidation(receipt)
	run.AddView(NewViewBuilder(ledger).Build(PlannerOptions{UseProjected: false}))
	return run
}

func scenarioCapacityCycle() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := newScenarioRun("capacity-cycle", ledger)
	builder := NewViewBuilder(ledger)
	run.AddView(builder.Build(PlannerOptions{UseProjected: false}))
	first, err := ledger.OpenSettlement("route-alpha-beta", 700_000, ReservationAllocation, false)
	if err == nil {
		run.Submitted = append(run.Submitted, first)
	}
	ledger.Advance(1)
	run.AddView(builder.Build(PlannerOptions{UseProjected: true}))
	ledger.Advance(2)
	run.AddCompleted(ledger.CompleteDueSettlements())
	run.AddView(builder.Build(PlannerOptions{UseProjected: true}))
	run.Note("capacity view follows observed and scheduled liquidity states")
	return run
}

func scenarioOperatorCycle() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := newScenarioRun("operator-cycle", ledger)
	allocator := NewAllocator(ledger)
	submitted, _ := allocator.ExecuteObserved(100_000)
	run.Submitted = append(run.Submitted, submitted...)
	ledger.Advance(2)
	run.AddCompleted(ledger.CompleteDueSettlements())
	engine := NewRebalanceEngine(ledger)
	result := engine.ExecuteCycle("operator-a", "post-settlement route refresh", false)
	run.AddRebalance(result)
	ledger.Advance(4)
	run.AddCompleted(ledger.CompleteDueSettlements())
	liquidator := NewLiquidationEngine(ledger)
	run.AddLiquidation(liquidator.Execute())
	run.AddView(NewViewBuilder(ledger).Build(PlannerOptions{UseProjected: false}))
	run.Note("operator cycle completed allocation, refresh and recovery passes")
	return run
}
