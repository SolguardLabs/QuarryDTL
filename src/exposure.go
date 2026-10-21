package main

import "sort"

// RouteExposure is an operator-facing aggregation of a route's current book.
type RouteExposure struct {
	RouteID        string `json:"route_id"`
	Asset          string `json:"asset"`
	Demand         Amount `json:"demand"`
	Allocated      Amount `json:"allocated"`
	Outstanding    Amount `json:"outstanding"`
	Submitted      Amount `json:"submitted"`
	Finalized      Amount `json:"finalized"`
	Failed         Amount `json:"failed"`
	FeesFinalized  Amount `json:"fees_finalized"`
	UtilizationBps int64  `json:"utilization_bps"`
}

func BuildRouteExposures(ledger *Ledger) []RouteExposure {
	byRoute := make(map[string]*RouteExposure)
	for _, route := range ledger.Routes.List() {
		utilization := int64(0)
		if route.Demand > 0 {
			utilization = int64(route.Allocated) * BasisPoint / int64(route.Demand)
		}
		byRoute[route.ID] = &RouteExposure{
			RouteID: route.ID, Asset: route.Asset, Demand: route.Demand,
			Allocated: route.Allocated, Outstanding: route.Outstanding,
			UtilizationBps: utilization,
		}
	}
	for _, settlement := range ledger.Settlements.List() {
		exposure, ok := byRoute[settlement.Route]
		if !ok {
			continue
		}
		switch settlement.Status {
		case SettlementOpen, SettlementPrepared, SettlementSubmitted:
			exposure.Submitted += settlement.Amount
		case SettlementFinalized:
			exposure.Finalized += settlement.Amount
			exposure.FeesFinalized += settlement.Fee
		case SettlementFailed:
			exposure.Failed += settlement.Amount
		}
	}
	out := make([]RouteExposure, 0, len(byRoute))
	for _, exposure := range byRoute {
		out = append(out, *exposure)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RouteID < out[j].RouteID })
	return out
}
