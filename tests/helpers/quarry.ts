import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const root = resolve(fileURLToPath(new URL("../..", import.meta.url)));
const exeName = process.platform === "win32" ? "quarrydtl.exe" : "quarrydtl";
const defaultBin = join(root, "out", exeName);

export type AmountBucket = {
  asset: string;
  amount: number;
};

export type VaultReport = {
  id: string;
  asset: string;
  reserve: number;
  idle: number;
  reserved: number;
  in_flight: number;
  liability: number;
  pending_in: number;
  pending_out: number;
  min_buffer: number;
  status: string;
};

export type RouteReport = {
  id: string;
  source_vault: string;
  target_vault: string;
  asset: string;
  demand: number;
  outstanding: number;
  allocated: number;
  settled: number;
  failed: number;
};

export type SettlementReport = {
  id: string;
  route: string;
  source_vault: string;
  target_vault: string;
  asset: string;
  amount: number;
  fee: number;
  status: string;
};

export type QuarryReport = {
  lab: string;
  scenario: string;
  network_id: string;
  clock: number;
  state_digest: string;
  assets: Array<Record<string, unknown>>;
  vaults: VaultReport[];
  routes: RouteReport[];
  accounts: Array<Record<string, unknown>>;
  settlements: SettlementReport[];
  reservations: Array<Record<string, unknown>>;
  forecasts: Array<Record<string, any>>;
  views: Array<Record<string, any>>;
  rebalances: Array<Record<string, any>>;
  liquidations: Array<Record<string, any>>;
  totals: {
    reserves: AmountBucket[];
    liabilities: AmountBucket[];
    pending_in: AmountBucket[];
    pending_out: AmountBucket[];
    fees: AmountBucket[];
  };
  risk: Record<string, number>;
  invariants: Record<string, boolean>;
  events: Array<Record<string, any>>;
  notes: string[];
};

export function binaryPath(): string {
  return process.env.QUARRY_BIN ?? defaultBin;
}

export function ensureBuilt(): void {
  if (process.env.QUARRY_BIN) return;
  if (existsSync(defaultBin)) return;
  const result = spawnSync(process.execPath, ["scripts/build.mjs"], {
    cwd: root,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (result.status !== 0) {
    throw new Error(["build failed", result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n"));
  }
}

export function runBinary(args: string[], expectSuccess = true) {
  ensureBuilt();
  const result = spawnSync(binaryPath(), args, {
    cwd: root,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (expectSuccess && result.status !== 0) {
    throw new Error(
      [`quarrydtl ${args.join(" ")} failed`, result.stdout.trim(), result.stderr.trim()]
        .filter(Boolean)
        .join("\n"),
    );
  }
  return result;
}

export function listScenarios(): string[] {
  return runBinary(["--list"]).stdout.trim().split(/\r?\n/).filter(Boolean);
}

export function runScenario(name: string): QuarryReport {
  const result = runBinary(["scenario", name]);
  return JSON.parse(result.stdout) as QuarryReport;
}

export function validateScenario(name: string): string {
  return runBinary(["validate", name]).stdout.trim();
}

export function byId<T extends { id: string }>(items: T[], id: string): T {
  const found = items.find((item) => item.id === id);
  assert.ok(found, `missing id ${id}`);
  return found;
}

export function bucket(entries: AmountBucket[], asset: string): number {
  const found = entries.find((entry) => entry.asset === asset);
  assert.ok(found, `missing asset ${asset}`);
  return found.amount;
}

export function events(report: QuarryReport, kind: string): Array<Record<string, any>> {
  return report.events.filter((event) => event.kind === kind);
}

export function assertDigest(value: unknown): void {
  assert.equal(typeof value, "string");
  assert.match(value as string, /^[0-9a-f]{32}$/);
}

export function assertCommon(report: QuarryReport, scenario: string): void {
  assert.equal(report.lab, "QuarryDTL");
  assert.equal(report.scenario, scenario);
  assert.equal(report.network_id, "quarry-local-liquidity");
  assertDigest(report.state_digest);
  assert.ok(report.assets.length >= 3);
  assert.ok(report.vaults.length >= 6);
  assert.ok(report.routes.length >= 4);
  assert.ok(Array.isArray(report.settlements));
  assert.ok(Array.isArray(report.forecasts));
  assert.ok(Array.isArray(report.events));
  assert.equal(report.invariants.vaults_non_negative, true);
  assert.equal(report.invariants.accounts_non_negative, true);
  assert.equal(report.invariants.settlement_links_valid, true);
  assert.equal(report.invariants.forecast_links_valid, true);
  assert.equal(report.invariants.routes_within_demand, true);
  assert.equal(report.invariants.capacity_views_non_negative, true);
  assert.equal(report.invariants.withdrawals_open, true);
}
