#!/usr/bin/env python3
"""List govulncheck findings that are NOT in the Go standard library.

Reads the govulncheck JSON stream on stdin and prints one line per
finding whose vulnerable module is something other than "stdlib", so the
release gate can separate "this module's dependencies have a
vulnerability" (fails the gate) from "the build toolchain has one"
(reported, toolchain-owned).

A finding is "called" when a vulnerable symbol is reachable; only those
are printed, which matches govulncheck's default text output.
"""

import json
import sys


def called_findings(text):
    """Yield every called finding in a govulncheck JSON document.

    The stream is a concatenation of JSON objects, not one object per
    line. A called finding carries a call trace longer than one frame;
    a finding whose trace has one frame only means the package is
    imported but no vulnerable symbol is reached.
    """
    decoder = json.JSONDecoder()
    pos = 0
    while pos < len(text):
        while pos < len(text) and text[pos].isspace():
            pos += 1
        if pos >= len(text):
            break
        event, pos = decoder.raw_decode(text, pos)
        finding = event.get("finding") if isinstance(event, dict) else None
        if finding and len(finding.get("trace") or []) > 1:
            yield finding


def non_stdlib(text):
    found = []
    for finding in called_findings(text):
        module = (finding.get("trace") or [{}])[0].get("module", "")
        if module == "stdlib":
            continue
        key = f"{finding.get('osv', '?')}: {module}"
        if key not in found:
            found.append(key)
    return found


def main():
    for line in non_stdlib(sys.stdin.read()):
        print(line)


if __name__ == "__main__":
    main()
