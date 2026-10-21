#!/usr/bin/env bash
set -euo pipefail

GO_BIN="${GO_BIN:-go}"
NODE_BIN="${NODE_BIN:-node}"

if ! command -v "$NODE_BIN" >/dev/null 2>&1 && command -v node.exe >/dev/null 2>&1; then
  NODE_BIN="node.exe"
fi

if command -v gofmt >/dev/null 2>&1; then
  GOFMT_BIN="gofmt"
else
  GO_ROOT="$("$GO_BIN" env GOROOT)"
  if command -v cygpath >/dev/null 2>&1 && [[ "$GO_ROOT" == [A-Za-z]:* || "$GO_ROOT" == *\\* ]]; then
    GO_ROOT="$(cygpath -u "$GO_ROOT")"
  fi
  GOFMT_BIN="$GO_ROOT/bin/gofmt"
fi

UNFORMATTED="$("$GOFMT_BIN" -l src)"
if [ -n "$UNFORMATTED" ]; then
  echo "$UNFORMATTED"
  echo "gofmt required"
  exit 1
fi

"$GO_BIN" vet ./...
"$GO_BIN" test ./...
"$NODE_BIN" scripts/build.mjs
"$NODE_BIN" --test --experimental-strip-types "tests/node/*.test.ts"
