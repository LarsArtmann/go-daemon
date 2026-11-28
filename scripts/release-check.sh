#!/usr/bin/env bash
# One command, one verdict: the release gate runs exactly the checks CI runs
# on a push, in CI's order, so a red push is caught locally first:
#
#   1. actionlint          — workflow syntax
#   2. go build            — every package compiles
#   3. check-readme.sh     — the README quickstart compiles against the library
#   4. go vet
#   5. go test -race       — plus the coverage gate (80% of statements)
#   6. govulncheck         — fails on any vulnerability in this module's own
#                            dependencies; stdlib findings are reported but
#                            attributed to the toolchain (see below)
#   7. golangci-lint
#   8. nix build + nix flake check --no-build --all-systems
#   9. check-docs.sh       — dprint, table rows, strikes, FEATURES counts
#  10. docs-gates run .    — freshness, annotate, routing, cadence, vocabulary
#
# Canonical invocation: `nix develop -c release-check` (supplies every tool
# on CI's versions; `nix run` would not work, because it chdirs into the
# store copy of the flake). Run from the repo root.
#
# Stdlib findings: govulncheck also reports vulnerabilities in the Go
# standard library, which belong to the toolchain the build uses, not to
# this module. The nixpkgs toolchain can trail the Go patch release that
# fixes one (1.27.3 carried CVE-2026-97054). Such a finding is printed and
# does not fail the gate; a finding in any other module does. When the
# toolchain is current the scan is clean and the distinction is moot.
set -euo pipefail

# Run in the caller's working directory: the script is installed into the
# nix store by the devShell, so its own location says nothing about the
# checkout under test.
[ -f flake.nix ] && [ -f go.mod ] || {
	echo "release-check: run from the root of the go-daemon checkout (flake.nix and go.mod not found in $(pwd))" >&2
	exit 2
}

coverage_threshold="80.0"
docs_gates_rev="github:Artmann-Technologies/docs-gates/bfdefbaf73d668fd4b2aab787ddecee7dc046d56"
coverage_dir="$(mktemp -d)"
trap 'rm -rf -- "$coverage_dir"' EXIT

failed=()
step() {
	echo
	echo "==> $1"
}

finish_step() {
	echo "    ok: $1"
}

run_step() {
	local name="$1"
	shift

	step "$name"
	if "$@"; then
		finish_step "$name"
	else
		echo "    FAILED: $1" >&2
		failed+=("$name")
	fi
}

step "actionlint"
if command -v actionlint >/dev/null 2>&1; then
	if actionlint -no-color; then
		finish_step "actionlint"
	else
		failed+=("actionlint")
	fi
else
	echo "    actionlint not found — run via 'nix develop -c release-check'" >&2
	failed+=("actionlint")
fi

run_step "go build" go build ./...
run_step "README quickstart" bash scripts/check-readme.sh
run_step "go vet" go vet ./...

step "go test -race"
if go test ./... -race -count=1 -covermode=atomic -coverprofile="$coverage_dir/coverage.out"; then
	finish_step "go test -race"
else
	echo "    FAILED: go test -race" >&2
	failed+=("go test -race")
fi

step "coverage gate (>= ${coverage_threshold}%)"
total="$(go tool cover -func="$coverage_dir/coverage.out" | awk '/^total:/ { gsub(/%/, "", $3); print $3 }')"
echo "    total statement coverage: ${total}%"
if awk -v total="$total" -v threshold="$coverage_threshold" \
	'BEGIN { exit !(total + 0 >= threshold + 0) }'; then
	finish_step "coverage gate"
else
	echo "    coverage ${total}% is below the ${coverage_threshold}% gate" >&2
	failed+=("coverage gate")
fi

step "govulncheck"
if govulncheck ./... >"$coverage_dir/vuln.txt" 2>&1; then
	finish_step "govulncheck"
else
	cat "$coverage_dir/vuln.txt"
	non_stdlib="$(
		govulncheck -json ./... 2>/dev/null |
			python3 scripts/govulncheck-nonstdlib.py
	)"
	if [ -n "$non_stdlib" ]; then
		echo "    vulnerabilities in this module's dependencies:" >&2
		echo "$non_stdlib" >&2
		failed+=("govulncheck")
	else
		echo "    reported findings are in the Go standard library (the build toolchain), not in this module" >&2
		echo "    stdlib findings do not fail this gate; bump the toolchain to clear them" >&2
	fi
fi

run_step "golangci-lint" golangci-lint run ./...

if command -v nix >/dev/null 2>&1; then
	run_step "nix build .#default" nix build .#default --no-write-lock-file --print-build-logs
	run_step "nix flake check" nix flake check --no-build --all-systems --no-write-lock-file
else
	echo "    nix not found; flake gates skipped" >&2
	failed+=("nix build .#default")
fi

run_step "documentation health (check-docs)" bash scripts/check-docs.sh

# The docs-health battery is not a flake input on purpose: its own inputs
# would triple flake.lock and it publishes no aarch64-darwin package. The
# rev-pinned run below is the same battery in every checkout and in CI.
# Bump the rev deliberately; the fleet default's marker vocabulary moves.
if command -v docs-gates >/dev/null 2>&1; then
	run_step "documentation health (docs-gates)" docs-gates run .
elif command -v nix >/dev/null 2>&1; then
	run_step "documentation health (docs-gates)" nix run "${docs_gates_rev}" --no-write-lock-file -- run .
else
	echo "    neither docs-gates nor nix found" >&2
	failed+=("documentation health (docs-gates)")
fi

echo
if [ "${#failed[@]}" -eq 0 ]; then
	echo "release-check: ALL GATES GREEN"
	exit 0
fi

echo "release-check: ${#failed[@]} gate(s) failed:" >&2
for name in "${failed[@]}"; do
	echo "  - $name" >&2
done
exit 1
