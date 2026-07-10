import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, byId, runScenario } from "../helpers/quarry.ts";

test("capacity cycle exposes observed and scheduled liquidity states", () => {
  const report = runScenario("capacity-cycle");
  assertCommon(report, "capacity-cycle");
  assert.equal(report.settlements.length, 1);
  assert.equal(report.settlements[0].status, "finalized");
  assert.ok(report.views.length >= 3);

  const middle = report.views[1].profiles as Array<Record<string, any>>;
  const beta = middle.find((profile) => profile.vault_id === "vault-beta-usdc");
  assert.ok(beta);
  assert.equal(beta.mode, "projected");
  assert.ok(beta.forecast > 0);

  const route = byId(report.routes, "route-alpha-beta");
  assert.equal(route.settled, 700_000);
});

test("operator cycle composes allocation, rebalance and recovery passes", () => {
  const report = runScenario("operator-cycle");
  assertCommon(report, "operator-cycle");
  assert.ok(report.risk.finalized_settlements >= 2);
  assert.ok(report.risk.submitted_rebalances >= 1);
  assert.ok(report.risk.liquidation_recoveries > 0);
});
