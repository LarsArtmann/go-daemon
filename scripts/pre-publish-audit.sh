#!/usr/bin/env bash
# Pre-publish audit: the three trust scans every repo flip needs, in one
# command. Generalizes the 2026-10-01 go-daemon pre-flip pass:
#   1. History secrets — gitleaks over every commit, plus a manual-pattern
#      pass for things generic rules miss (internal hosts/paths,
#      .netrc-shaped content, long high-entropy strings).
#   2. Working-tree private paths — references to /home, GOPRIVATE,
#      git+ssh pins, and SSH remotes that would leak or break once public.
#   3. License sanity — every module in the build closure must carry a
#      permissive license file (MIT/BSD/Apache-2.0/MPL); copyleft fails.
#
# Exit 0 = publishable from a trust perspective (owner decisions aside).
# Exit 1 = findings listed above the exit. Run from the repo to audit.
set -uo pipefail

fail=0

echo "== 1/3 git-history secret scan"
if command -v gitleaks >/dev/null 2>&1; then
	if ! gitleaks git --no-banner -v . 2>&1 | tail -2; then
		fail=1
	fi
else
	echo "pre-publish: gitleaks not installed — install it or run via nix shell nixpkgs#gitleaks" >&2
	fail=1
fi

echo "== 1b/3 manual credential-pattern pass over all blobs"
mapfile -t oids < <(git rev-list --objects --all | awk 'NF>1{print $1}' | sort -u)
for oid in "${oids[@]}"; do
	hits=$(git cat-file blob "$oid" 2>/dev/null | grep -niE \
		'/home/[a-z]+|git@github|ssh://|\.netrc|machine .+ login|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY|xox[baprs]-|gh[pousr]_[A-Za-z0-9]{36}|AIza[0-9A-Za-z_-]{35}|[A-Za-z0-9+/]{60,}={0,2}' || true)
	if [ -n "$hits" ]; then
		path=$(git rev-list --objects --all | awk -v o="$oid" '$1==o{print $2; exit}')
		echo "MANUAL-PATTERN hit in blob $oid ($path):"
		echo "$hits" | head -5
	fi
done

echo "== 2/3 working-tree private-path scan"
if grep -rnE 'GOPRIVATE|git\+ssh|/home/[a-z]+' --include='*.nix' --include='*.go' --include='*.mod' --include='*.yml' --include='*.md' . 2>/dev/null | grep -v '^\.git' | grep -vE 'docs/(status|planning)/|scripts/pre-publish-audit.sh|CHANGELOG'; then
	echo "pre-publish: private-path references above — de-GOPRIVATE or scrub before flipping" >&2
	fail=1
else
	echo "no private-path references outside historical docs."
fi

echo "== 3/3 build-closure license scan"
if command -v go >/dev/null 2>&1; then
	MC=$(go env GOMODCACHE)
	while read -r mod; do
		[ -z "$mod" ] && continue
		ver=$(go list -m -f '{{.Version}}' "$mod" 2>/dev/null) || continue
		# The root module (this repo) has no version in the module graph;
		# its license is the repo's own LICENSE file.
		[ -z "$ver" ] && continue
		f=$(ls "$MC/$mod@$ver"/LICENSE* 2>/dev/null | head -1)
		t=$(cat "$f" 2>/dev/null)
		case "$t" in
		*"Apache License"* | *"Permission is hereby granted"* | *"Redistributions of source code"* | *"redistributions of source code"* | *"Redistribution and use in source and binary forms"* | *"Mozilla Public License"*) ;;
		*)
			echo "pre-publish: $mod@$ver has no recognizable permissive license file"
			fail=1
			;;
		esac
	done < <(go list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' ./... 2>/dev/null | sort -u)
	echo "build-closure licenses scanned."
else
	echo "pre-publish: go not installed — license scan skipped" >&2
	fail=1
fi

if [ "$fail" -eq 0 ]; then
	echo "pre-publish audit: CLEAN (owner decisions aside)."
fi

exit "$fail"
