#!/usr/bin/env bash
set -euo pipefail

# Cross-repository smoke test: build real weave and weave-code binaries, run
# project analysis, and verify declared document types and _code artifacts.

cd "$(dirname "$0")/.."
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

weave_src="${WEAVE_SOURCE:-}"
if [ -z "$weave_src" ]; then
  weave_ref="${WEAVE_REF:-main}"
  git clone --quiet --depth 1 --branch "$weave_ref" https://github.com/prulloac/weave.git "$work/weave-src"
  weave_src="$work/weave-src"
fi

mkdir -p "$work/bin"
(cd "$weave_src" && go build -o "$work/bin/weave" ./cmd/weave)
go build -o "$work/bin/weave-code" .
export PATH="$work/bin:$PATH"
export WEAVE_CODE_SYMBOL_DOCUMENTS=1

mkdir -p "$work/src"
cat > "$work/src/main.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("integration")
}
EOF

result="$("$work/bin/weave" ingest "$work/src" --with-enricher code --db "$work/index.db" --json)"
printf '%s\n' "$result"
printf '%s' "$result" | grep -q '"ok":true'

code="$("$work/bin/weave" list "$work/src" --type Code --db "$work/index.db" --json)"
printf '%s\n' "$code"
printf '%s' "$code" | grep -q 'main.go'

"$work/bin/weave" export "$work/src" --out "$work/export" --format raw --yes --db "$work/index.db" --json
test -f "$work/export/_code/symbols.json"
grep -q 'weave.code-symbols/v1' "$work/export/_code/symbols.json"
grep -q 'main' "$work/export/_code/symbols.json"

echo "integration: ok"
