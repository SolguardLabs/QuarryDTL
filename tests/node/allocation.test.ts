import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, byId, bucket, events, runScenario } from "../helpers/quarry.ts";

test("observed allocation settles idle reserves into demand routes", () => {
  const report = runScenario("allocation");
  assertCommon(report, "allocation");
  assert.ok(report.settlements.length >= 2);
  assert.ok(report.settlements.every((settlement) => settlement.status === "finalized"));
  assert.ok(events(report, "allocation.executed").length >= 2);

  const alphaBeta = byId(report.routes, "route-alpha-beta");
  assert.equal(alphaBeta.outstanding, 0);
  assert.ok(alphaBeta.settled > 0);

  const usdcReserve = bucket(report.totals.reserves, "usdc");
  assert.ok(usdcReserve > 0);
  assert.ok(report.risk.protocol_fees_collected > 0);
});

test("allocation preserves vault buffers after settlement", () => {
  const report = runScenario("allocation");
  const alpha = byId(report.vaults, "vault-alpha-usdc");
  assert.ok(alpha.idle >= alpha.min_buffer);
  assert.ok(alpha.reserve >= alpha.liability);

  const north = byId(report.vaults, "vault-north-usdt");
  assert.ok(north.reserve > 260_000);
  assert.equal(report.totals.pending_in.find((entry) => entry.asset === "usdc")?.amount, 0);
});
