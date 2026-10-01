#!/usr/bin/env bash
# Compiles the first ```go fence of README.md (the quickstart) as a package
# main inside this module, so a broken example fails the build instead of
# shipping as a history blemish (the caf1f0c class). The second fence is an
# intentional fragment and is deliberately not extracted.
#
# Single source of truth for this gate: the flake's check-readme app invokes
# this script, and CI runs it directly — the two can never drift.
set -euo pipefail

# Run from the repo root (the CI checkout dir, or the directory you call
# `nix run .#check-readme` from): README.md and go.mod live there.

[ -f README.md ] || {
	echo "check-readme: run from the repo checkout (README.md not found)" >&2
	exit 1
}

work=".readmecheck"
trap 'chmod -R u+w "$work" 2>/dev/null; rm -rf "$work"' EXIT

mkdir -p "$work"

awk 'BEGIN{inblock=0} /^```go$/{inblock=1; next} /^```$/{if(inblock) exit} inblock{print}' README.md >"$work/main.go"

if [ ! -s "$work/main.go" ]; then
	echo "check-readme: no go code fence found in README.md" >&2
	exit 1
fi

go build -o /dev/null "./$work"
echo "README quickstart snippet compiles."
