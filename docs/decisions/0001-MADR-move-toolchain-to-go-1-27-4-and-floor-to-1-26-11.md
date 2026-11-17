---
status: accepted
date: 2026-11-17
decision-makers: the repository owner
consulted: "govulncheck v1.9.0 (database updated 2026-11-11) under Go 1.26.11 and 1.27.4, 2026-11-17; golangci-lint v2.13.1 and v2.15.0 built into a scratch directory; go-selfupdate-lib's 0045 record pair, read-only"
informed: nobody else requires this module (see Consequences)
---
# Build with Go 1.27.4 through the `toolchain` line, raise the floor to `go 1.26.11`, and move CI's linter to v2.15.0

## Context and Problem Statement

This is the first decision record in this repository. It sizes the fleet's
Go-pins decision, first made in go-selfupdate-lib's
`0045-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md`, to this
repository.

The owner's instruction of 2026-11-17, relayed by the coordinator, verbatim:

> "go-llmprovider-sdk, go-tui-lib, acp-go-sdk2, gobble-cli, magic-cli-remote, go-selfupdate-lib, mcp-server-duckduckgo, mcp-server-magictools, mcp-server-recall, mcp-server-sequential-thinking, mcplib, mcp-buntdb, and mcp-server-socratic-thinker all need to have their go floor pinned to 1.26.11, and go version/toolchain pinned to 1.27.4."

"Floor" is the `go` directive in `go.mod`; "version/toolchain" is the
`toolchain` line. A read-only survey then found that this repository builds
under 1.26.11. The owner ruled, labels exact: "One short record pair per
repository, all approved now through CI (Recommended)", and on the linter,
which breaks under Go 1.27.4: "v2.15.0 everywhere, in each repository's
Go-pins record (Recommended)", with any new finding the newer linter reports
brought to the owner with the code rather than suppressed.

Why now: GO-2026-6859, in `net/http`, is reported by `govulncheck` v1.9.0
(database of 2026-11-11) under Go 1.27.3 and 1.26.10 and not under 1.27.4
and 1.26.11 (measured by the coordinator, 2026-11-17). This repository's
floor of 1.26.6 is below the fixed 1.26.11.

**Evidence index.** Read at HEAD `ef5ba25` (branch `main`, clean, level with
its upstream ref) on 2026-11-17 unless stated.

* **E1, the pins.**
  * `go.mod:3`: `go 1.26.6`; there is no `toolchain` line.
  * `Makefile:1`: `MOD_VERSION := 1.26.6`. The variable is defined and
    referenced nowhere else in the repository (whole-tree search): it is
    dead, and it is a stated version, so it moves with the floor.
  * `.github/workflows/ci.yml:15` and `:33`: `actions/setup-go@v7` with
    `go-version-file: go.mod`.
  * `.github/workflows/ci.yml:20`: `golangci-lint/v2/cmd/golangci-lint@v2.13.1`.
  * `README.md` and every other file: no statement of a Go version (search
    for `go 1.`, `golang`, `1.2x`: only "Built with Go." at `README.md:451`).
* **E2, records.** No `NNNN-{MADR,PLAN,REPORT,GATES}-*.md` exists anywhere in
  the repository, there is no `docs/` directory, and no records tooling or
  index. This pair is `0001`, numbered by a whole-repository scan.
* **E3, CI's Go.** Per the README of `actions/setup-go` as quoted in
  go-selfupdate-lib's `0018-MADR-move-toolchain-to-go-1-27-2.md` (v7.0.0): "If
  the `toolchain` directive is present, its version is used; otherwise, the
  action falls back to the `go` directive." So after this change CI's two
  `setup-go` steps install Go 1.27.4; today, with no toolchain line, they
  install 1.26.6. CI has no job that builds on the floor (see Consequences).
* **E4, the linter [observed].** golangci-lint v2.13.1 (built with Go 1.26.11)
  and v2.15.0 (built with Go 1.27.4) were each run with `make lint` on a
  scratch copy of HEAD under Go 1.26.6: both report `0 issues.`, exit 0. The
  v2.13.1 binary cannot type-check code under Go 1.27.4 (owner-relayed
  finding from another repository: "export data version 5 is greater than
  maximum supported version 4"), which is why CI's pin moves with the
  toolchain.
* **E5, the module graph.** `go list -m all` under 1.26.11 (survey): every
  dependency declares a floor at or below 1.26.5; `mcplib v1.2.0` declares
  `go 1.26.5`. None needs raising.

## Considered Options

* A. Floor `go 1.26.11`, `toolchain go1.27.4`, CI linter v2.15.0 (as the
  owner ruled).
* B. A different floor (keep 1.26.6, or take 1.27.4 as the floor).
* C. Change nothing.

## Decision Outcome

Chosen option: "A. Floor `go 1.26.11`, `toolchain go1.27.4`, CI linter
v2.15.0", because the owner ruled it, it clears GO-2026-6859 in every build
this repository makes, and the only linter that can read Go 1.27.4's export
data is the one that goes with it.

### Consequences

* Good, because CI builds and ships binaries with 1.27.4, which
  `govulncheck` reports clean.
* Good, because the floor stops at the last 1.26 patch with the fix, so
  1.26 users are not forced onto 1.27.
* Bad, because the floor rises by five patch releases for anyone building
  from source with a plain 1.26 toolchain older than 1.26.11; the Go command
  fetches 1.26.11 or 1.27.4 on demand unless `GOTOOLCHAIN=local`.
* Bad (a named gap), because CI has no job that builds and tests on the
  floor: after this change CI exercises only 1.27.4. The floor's build and
  tests are run once, by hand, in the PLAN's gates. Whether to add floor
  jobs is the owner's decision, fleet-wide, and is not made here.
* Neutral, because this repository is a server binary: **consumers.** No
  other fleet module requires it (a search of every `go.mod` under the
  fleet's Go checkouts finds only this module naming itself). Its users run
  the released binary, built by CI on a `v*` tag. An installed binary
  benefits only from a new release built after this change; the latest tag
  is `v1.0.2`, and `HEAD` is four commits past it. A tag and release need
  the owner's word and are outside the PLAN.
* The `Makefile`'s dead `MOD_VERSION` is kept and moved to `1.26.11`, so no
  statement of a version in the tree is false; removing it is a separate
  cleanup.

### Confirmation

* The PLAN's version-agreement check passes on the changed tree and fails on
  three planted wrong versions.
* `go mod tidy -diff` shows nothing; `go.sum` is unchanged.
* `govulncheck ./...` exits 0 under `GOTOOLCHAIN=go1.27.4` and under
  `GOTOOLCHAIN=go1.26.11`.
* `make lint` with a scratch-built v2.15.0 reports no issue under both
  toolchains.
* On the pushed commit, CI's `setup-go` logs name Go 1.27.4 (checked by the
  coordinator, not here).

## Pros and Cons of the Options

### A. As ruled

* Good, because it is the owner's ruling and it is confirmed by the checks
  above.
* Bad, because of the floor rise and the missing floor job (Consequences).

### B. A different floor

* Good, because keeping 1.26.6 would change nothing for source builders.
* Bad, because 1.26.6 is below the 1.26.11 that fixes GO-2026-6859 (the
  coordinator's measurement, Context), so a build on the floor would fail `govulncheck`; the owner named 1.26.11.
* Bad, because a floor of 1.27.4 would force 1.27 on every source builder
  for no code reason (E5; the code builds on 1.26.11).

### C. Change nothing

* Good, because no file changes.
* Bad, because CI keeps building released binaries with Go 1.26.6, which
  carries the advisory, and the owner has ruled otherwise.

## More Information

* **Earlier records.** None in this repository. The prior floor commit,
  `ef5ba25` ("build: raise the Go floor to 1.26.6"), is history and is not
  amended.
* **Release.** Not part of this decision. The PLAN stops before any tag.
* **Not verified.**
  * That `actions/setup-go@v7` (a floating major tag) installs 1.27.4 on the
    runners: the quoted README is of v7.0.0 and no CI run has happened.
  * That CI's `go install ...@v2.15.0` builds on the runner's Go 1.27.4 and
    passes: only a local build was observed.
  * What GO-2026-6859 is beyond its title and fixed versions.
  * Behaviour of the code under 1.27.4's other changes beyond the test
    suite and `govulncheck`.
