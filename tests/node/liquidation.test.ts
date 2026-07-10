import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, byId, events, runScenario } from "../helpers/quarry.ts";

test("liquidation cycle recovers under-covered vault accounting", () => {
  const report = runScenario("liquidation");
  assertCommon(report, "liquidation");
  assert.equal(report.liquidations.length, 1);
  assert.ok(report.liquidations[0].candidates.length >= 1);
  assert.ok(report.liquidations[0].recovered > 0);
  assert.ok(events(report, "liquidation.executed").length, "liquidation event missing");

  const gamma = byId(report.vaults, "vault-gamma-usdc");
  assert.ok(gamma.liability < 900_000);
  assert.ok(gamma.reserve < 650_000);
});
