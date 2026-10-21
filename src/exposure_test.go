package main

import "testing"

func TestRouteExposureLifecycle(t *testing.T) {
	ledger := BuildSeedLedger()
	initial := BuildRouteExposures(ledger)
	if len(initial) != 4 {
		t.Fatalf("expected four routes, got %d", len(initial))
	}
	for _, route := range initial {
		if route.Allocated != 0 || route.UtilizationBps != 0 {
			t.Fatalf("unexpected initial exposure: %+v", route)
		}
	}
	id, err := ledger.OpenSettlement("route-alpha-beta", 700_000, ReservationAllocation, false)
	if err != nil {
		t.Fatal(err)
	}
	pending := BuildRouteExposures(ledger)
	if pending[0].Submitted != 700_000 || pending[0].UtilizationBps != 10_000 {
		t.Fatalf("incorrect submitted view: %+v", pending[0])
	}
	ledger.Advance(3)
	if err := ledger.CompleteSettlement(id); err != nil {
		t.Fatal(err)
	}
	final := BuildRouteExposures(ledger)
	if final[0].Submitted != 0 || final[0].Finalized != 700_000 || final[0].FeesFinalized <= 0 {
		t.Fatalf("incorrect finalized view: %+v", final[0])
	}
}

func TestRouteExposureFailure(t *testing.T) {
	ledger := BuildSeedLedger()
	id, err := ledger.OpenSettlement("route-alpha-beta", 700_000, ReservationAllocation, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ledger.FailSettlement(id, "route unavailable"); err != nil {
		t.Fatal(err)
	}
	view := BuildRouteExposures(ledger)[0]
	if view.Failed != 700_000 || view.Submitted != 0 || view.Finalized != 0 || view.FeesFinalized != 0 {
		t.Fatalf("incorrect failed view: %+v", view)
	}
}
