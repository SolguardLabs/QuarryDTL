import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, assertDigest, listScenarios, runBinary, runScenario, validateScenario } from "../helpers/quarry.ts";

test("CLI lists stable audit scenarios", () => {
  assert.deepEqual(listScenarios(), [
    "baseline",
    "allocation",
    "rebalance",
    "liquidation",
    "capacity-cycle",
    "operator-cycle",
  ]);
});

test("baseline scenario emits the public JSON contract", () => {
  const report = runScenario("baseline");
  assertCommon(report, "baseline");
  assert.equal(report.settlements.length, 0);
  assert.ok(report.totals.reserves.length >= 3);
  assert.ok(report.risk.total_demand > 0);
  assertDigest(report.state_digest);
});

test("validate command returns scenario digest", () => {
  const output = validateScenario("baseline");
  assert.match(output, /^ok baseline [0-9a-f]{32}$/);
});

test("unknown scenarios fail without JSON output", () => {
  const result = runBinary(["scenario", "missing-scenario"], false);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unknown scenario/);
  assert.equal(result.stdout.trim(), "");
});
