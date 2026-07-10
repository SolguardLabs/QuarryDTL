package main

import "fmt"

type ForecastStatus string

const (
	ForecastScheduled ForecastStatus = "scheduled"
	ForecastObserved  ForecastStatus = "observed"
	ForecastSettled   ForecastStatus = "settled"
	ForecastVoided    ForecastStatus = "voided"
)

type ForecastClass string

const (
	ForecastRouteInflow ForecastClass = "route_inflow"
	ForecastSweep       ForecastClass = "sweep"
	ForecastRecovery    ForecastClass = "recovery"
)

type Forecast struct {
	ID               string         `json:"id"`
	Vault            string         `json:"vault"`
	Route            string         `json:"route"`
	Asset            string         `json:"asset"`
	Amount           Amount         `json:"amount"`
	VisibleAt        int64          `json:"visible_at"`
	MaturesAt        int64          `json:"matures_at"`
	ConfidenceBps    int64          `json:"confidence_bps"`
	SourceSettlement string         `json:"source_settlement,omitempty"`
	Status           ForecastStatus `json:"status"`
	Class            ForecastClass  `json:"class"`
}

func NewForecast(id string, settlement Settlement) Forecast {
	return Forecast{
		ID:               id,
		Vault:            settlement.TargetVault,
		Route:            settlement.Route,
		Asset:            settlement.Asset,
		Amount:           settlement.Amount - settlement.Fee,
		VisibleAt:        settlement.OpenedAt,
		MaturesAt:        settlement.MaturesAt,
		ConfidenceBps:    9_700,
		SourceSettlement: settlement.ID,
		Status:           ForecastScheduled,
		Class:            ForecastRouteInflow,
	}
}

func (f Forecast) Visible(clock int64) bool {
	return f.Status == ForecastScheduled || f.Status == ForecastObserved || f.Status == ForecastSettled && clock >= f.VisibleAt
}

func (f Forecast) Assignable(clock int64, minimumConfidence int64) bool {
	if clock < f.VisibleAt {
		return false
	}
	if f.ConfidenceBps < minimumConfidence {
		return false
	}
	return f.Status == ForecastScheduled || f.Status == ForecastObserved || f.Status == ForecastSettled
}

func (f Forecast) Mature(clock int64) bool {
	return clock >= f.MaturesAt
}

func (f *Forecast) Observe() {
	if f.Status == ForecastScheduled {
		f.Status = ForecastObserved
	}
}

func (f *Forecast) Settle() {
	f.Status = ForecastSettled
}

func (f *Forecast) Void() {
	f.Status = ForecastVoided
}

type ForecastBook struct {
	forecasts map[string]*Forecast
	order     []string
}

func NewForecastBook() *ForecastBook {
	return &ForecastBook{forecasts: map[string]*Forecast{}, order: []string{}}
}

func (b *ForecastBook) Add(forecast Forecast) error {
	if forecast.ID == "" {
		return fmt.Errorf("forecast id missing")
	}
	if forecast.Amount < 0 {
		return fmt.Errorf("forecast %s amount cannot be negative", forecast.ID)
	}
	if _, ok := b.forecasts[forecast.ID]; !ok {
		b.order = append(b.order, forecast.ID)
	}
	copy := forecast
	b.forecasts[forecast.ID] = &copy
	return nil
}

func (b *ForecastBook) Get(id string) (*Forecast, bool) {
	forecast, ok := b.forecasts[id]
	return forecast, ok
}

func (b *ForecastBook) MustGet(id string) *Forecast {
	forecast, ok := b.Get(id)
	if !ok {
		panic("missing forecast " + id)
	}
	return forecast
}

func (b *ForecastBook) List() []Forecast {
	out := make([]Forecast, 0, len(b.order))
	for _, id := range b.order {
		out = append(out, *b.forecasts[id])
	}
	return out
}

func (b *ForecastBook) ForVault(vaultID string) []Forecast {
	out := []Forecast{}
	for _, id := range b.order {
		forecast := b.forecasts[id]
		if forecast.Vault == vaultID {
			out = append(out, *forecast)
		}
	}
	return out
}

func (b *ForecastBook) AssignableForVault(vaultID string, clock int64, minimumConfidence int64) []Forecast {
	out := []Forecast{}
	for _, id := range b.order {
		forecast := b.forecasts[id]
		if forecast.Vault == vaultID && forecast.Assignable(clock, minimumConfidence) {
			out = append(out, *forecast)
		}
	}
	return out
}

func (b *ForecastBook) Observe(clock int64) []string {
	observed := []string{}
	for _, id := range b.order {
		forecast := b.forecasts[id]
		if forecast.Status == ForecastScheduled && clock >= forecast.VisibleAt {
			forecast.Observe()
			observed = append(observed, id)
		}
	}
	return observed
}

func (b *ForecastBook) SettleBySource(settlementID string) {
	for _, forecast := range b.forecasts {
		if forecast.SourceSettlement == settlementID {
			forecast.Settle()
		}
	}
}

func (b *ForecastBook) VoidBySource(settlementID string) {
	for _, forecast := range b.forecasts {
		if forecast.SourceSettlement == settlementID {
			forecast.Void()
		}
	}
}

func (b *ForecastBook) TotalAssignable(vaultID string, clock int64, minimumConfidence int64) Amount {
	total := Amount(0)
	for _, forecast := range b.AssignableForVault(vaultID, clock, minimumConfidence) {
		total += forecast.Amount
	}
	return total
}
