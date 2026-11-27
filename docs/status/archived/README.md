# Archived status reports

Closed history. Every report here went through the docs-health archive ritual
(2026-11-27): numbered items and table rows carry inline verdicts, and the
still-open work is named in the `Live remainder` banner at the top of each
file and routed into the living docs. **Do not harvest archived reports** —
their open work already lives in `TODO_LIST.md`, `ROADMAP.md`, or the named
sibling repository.

## Manifest (bulk archive, 2026-11-27)

| File                                                                 | Classification | Deciding reason                                                                              |
| -------------------------------------------------------------------- | -------------- | -------------------------------------------------------------------------------------------- |
| `2026-09-29_09-01_go-daemon-adoption-verification-...closeout.md`    | ARCHIVE        | §a done; §f rows already routed/done; §g questions are consumer/owner calls                  |
| `2026-09-29_11-35_negotiation-qvalue-exactness-overhaul-status.md`   | ARCHIVE        | Every item resolved in place; questions answered by the later contract tests                 |
| `2026-09-30_00-43_docs-health-audit-negotiation-verification-...md`  | ARCHIVE        | §a done; §f residuals are cross-repo queues or answered owner decisions                      |
| `2026-09-30_10-33_go-public-session-review.md`                       | ARCHIVE        | The flip/license/name/visibility items executed 2026-11-27; the rest routed to ROADMAP       |
| `2026-09-30_11-29_release-v0.2.0-and-full-todo-closeout.md`          | ARCHIVE        | §a done and §f resolved; §g settled by later releases                                        |
| `2026-10-01_02-15_docs-health-annotate-archive-harvest-pass.md`      | ARCHIVE        | §a done; §b partials closed by the plan execution and the flip                               |
| `2026-10-01_03-52_pareto-plan-execution-trust-tests-ci.md`           | ARCHIVE        | Plan executed end to end; §f cross-repo/fleet backlog routed                                 |
| `2026-10-01_05-46_sse-test-dedup-session-status.md`                  | ARCHIVE        | §a done; clone/gate hygiene recorded in AGENTS.md and ROADMAP                                |
| `2026-10-01_05-51_erraudit-review-error-policy.md`                   | ARCHIVE        | §a done; automation residuals routed to ROADMAP theme 3                                      |
| `2026-10-01_07-05_sseparse-v0.2.0-release-and-go-daemon-adoption.md` | ARCHIVE        | §a done; go-sse items routed to that repo and ROADMAP                                        |
| `2026-10-01_09-59_v030-release-and-pdd-bump.md`                      | ARCHIVE        | §a done; §b-e routed; the v0.3.0 draft is annotated executed                                 |
| `2026-10-04_08-20_f2-jsonv2-write-path-injection.md`                 | ARCHIVE        | F2 closed; F3 lives in TODO_LIST; the v1-flavour claim is superseded by the json/v2 decision |
| `2026-10-05_12-28_branching-flow-panic-finding-review.md`            | ARCHIVE        | §a done; tool adoption and nil-codec hardening routed to ROADMAP/TODO_LIST                   |
| `2026-10-05_12-32_notify-loggerdefault-dedup-session-status.md`      | ARCHIVE        | §a done; nil-logger tests exist; the rest routed to TODO_LIST/ROADMAP                        |
| `2026-10-08_22-46_supervision-doc-session-status.md`                 | ARCHIVE        | §a/checklist done; supervision doc now covers memory limits and CONTRIBUTING links it        |

## Convention: the §f ledger

A §f "next steps" table in this corpus is a brainstorm ledger, not a backlog:
resolved rows are struck with a verdict, and the surviving rows stay unstruck
by design (absence of a marker is the open signal). The table is preceded by
a `check-rows:ledger` marker and a disposition banner naming where the open
work lives. That is why a file here can carry unstruck rows and still be
archived: the routing is explicit, not forgotten.
