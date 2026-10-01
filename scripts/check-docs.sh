#!/usr/bin/env bash
# One-command documentation health gate. Chains:
#   1. dprint check        — formatting of every markdown/json/yaml file
#   2. scripts/check-rows.py — per-table row completeness (no PARTIAL rows,
#      no mixed struck/unstruck tables) across the living docs and the
#      annotated historical reports
#   3. strike presence     — every historical report under docs/status/ and
#      docs/planning/archived/ must still carry its ~~ annotations (dprint's
#      markdown plugin has destroyed full-row strikes before)
#   4. FEATURES count gate — the "Computed from this file" line is derived
#      from the actual table rows; drift fails (the counts rotted once)
#
# Canonical invocation: `nix run .#check-docs` (which supplies dprint and
# python3). Run from the repo root.
set -euo pipefail

fail=0

[ -f FEATURES.md ] || {
	echo "check-docs: run from the repo root (FEATURES.md not found)" >&2
	exit 1
}

echo "== dprint check"
if command -v dprint >/dev/null 2>&1; then
	dprint check || fail=1
else
	echo "check-docs: dprint not found — run via 'nix run .#check-docs'" >&2
	fail=1
fi

echo "== check-rows"
script_dir="$(dirname "$0")"
# Living docs only: TODO_LIST/FEATURES/ROADMAP tables must be uniformly
# struck or uniformly open. Historical reports under docs/status/ contain
# deliberately-open rows frozen at annotation time (checked for strike
# PRESENCE below, not per-table completeness).
files=(TODO_LIST.md FEATURES.md ROADMAP.md)
if ! python3 "$script_dir/check-rows.py" "${files[@]}"; then
	fail=1
fi

echo "== strike presence in historical reports"
missing_strikes=0
for f in docs/status/*.md docs/planning/archived/*.md; do
	if [ -f "$f" ] && ! grep -q '~~' "$f"; then
		echo "check-docs: $f carries no strikethrough annotations"
		missing_strikes=$((missing_strikes + 1))
	fi
done
[ "$missing_strikes" -eq 0 ] || fail=1

echo "== FEATURES counts"
counts_para="$(awk '/^## Counts/{flag=1; next} flag && NF==0{if (seen) exit; next} flag{seen=1; printf "%s ", $0}' FEATURES.md)"
labels=(FULLY_FUNCTIONAL PARTIALLY_FUNCTIONAL BROKEN DISABLED PLANNED ON_HOLD)
for label in "${labels[@]}"; do
	claimed="$(grep -oE "[0-9]+ $label" <<<"$counts_para" | grep -oE '^[0-9]+' || echo 0)"
	actual="$(grep -cE "\| ${label}[[:space:]]+\|" FEATURES.md || true)"
	if [ "$claimed" != "$actual" ]; then
		echo "check-docs: FEATURES count drift for $label: claimed=$claimed actual=$actual"
		fail=1
	fi
done

if [ "$fail" -eq 0 ]; then
	echo "check-docs: all documentation gates pass."
fi

exit "$fail"
