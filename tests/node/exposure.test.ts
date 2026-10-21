import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, runScenario } from "../helpers/quarry.ts";

test("route exposure reconciles finalized amount and realized fees", () => {
  const report = runScenario("allocation");
  assertCommon(report, "allocation");
  const exposures = report.exposure;
  assert.equal(exposures.length, report.routes.length);
  const alpha = exposures.find((entry) => entry.route_id === "route-alpha-beta");
  assert.ok(alpha);
  assert.equal(alpha.demand, 700_000);
  assert.equal(alpha.finalized, 700_000);
  assert.equal(alpha.submitted, 0);
  assert.equal(alpha.utilization_bps, 10_000);
  assert.ok(alpha.fees_finalized > 0);
});

test("route exposure separates pending and finalized settlement amounts", () => {
  const report = runScenario("rebalance");
  assertCommon(report, "rebalance");
  const pending = report.exposure.filter((entry) => entry.submitted > 0);
  assert.ok(pending.length > 0);
  for (const entry of pending) {
    assert.equal(entry.finalized, 0);
    assert.equal(entry.fees_finalized, 0);
    assert.ok(entry.allocated <= entry.demand);
  }
});
