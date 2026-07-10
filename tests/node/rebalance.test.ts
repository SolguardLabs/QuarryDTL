import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, byId, events, runScenario } from "../helpers/quarry.ts";

test("rebalance uses forecast-aware capacity and finalizes route refresh", () => {
  const report = runScenario("rebalance");
  assertCommon(report, "rebalance");
  assert.equal(report.rebalances.length, 1);
  assert.equal(report.rebalances[0].mode, "projected");
  assert.ok(report.rebalances[0].accepted_amount >= 250_000);
  assert.ok(events(report, "rebalance.executed").length, "rebalance event missing");

  const betaDelta = byId(report.routes, "route-beta-delta");
  assert.ok(betaDelta.settled >= 250_000);
  assert.equal(betaDelta.outstanding, 0);

  const beta = byId(report.vaults, "vault-beta-usdc");
  assert.equal(beta.pending_in, 0);
  assert.equal(beta.pending_out, 0);
});

test("rebalance reports both observed and projected capacity views", () => {
  const report = runScenario("rebalance");
  assert.ok(report.views.length >= 2);
  const projectedView = report.views.find((view) =>
    (view.profiles as Array<Record<string, unknown>>).some((profile) => profile.mode === "projected"),
  );
  assert.ok(projectedView, "expected projected capacity profile");
});
