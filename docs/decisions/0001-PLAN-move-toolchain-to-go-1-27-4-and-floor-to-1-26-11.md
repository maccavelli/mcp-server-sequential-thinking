---
status: in-progress
date: 2026-11-17
associated-madr: "0001-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md"
---
# Implement Go 1.27.4 as the toolchain, `go 1.26.11` as the floor, and golangci-lint v2.15.0 in CI

Associated MADR: [0001-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md](0001-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md)
(option A, as the owner ruled).

## Goal

* `go.mod` says `go 1.26.11` and `toolchain go1.27.4`.
* Every current statement of a version in the tree agrees, including the CI
  linter pin.
* The gates pass and `govulncheck` exits 0 under both toolchains.
* The plan stops before any tag or release.

## Scope

### In scope

| Step | File | Change |
| :--- | :--- | :--- |
| 1 | `go.mod:3` | `go 1.26.6` becomes `go 1.26.11`; a `toolchain go1.27.4` line follows it |
| 2 | `Makefile:1` | `MOD_VERSION := 1.26.6` becomes `1.26.11` |
| 3 | `.github/workflows/ci.yml:20` | `golangci-lint@v2.13.1` becomes `@v2.15.0` |

### Out of scope

* Any tag or release; any push (the coordinator commits and pushes).
* A floor job in CI (a named gap in the MADR).
* Raising `mcplib` or any other dependency; `go.sum` must not move.
* Any lint finding: none is expected, and any that appears is reported to the
  owner, never suppressed or fixed here.
* History: the `ef5ba25` commit and all earlier commits.

## Implementation Steps

1. Apply steps 1 to 3 above with a script that asserts each old string occurs
   exactly once before replacing it; read each result back.
2. `go mod tidy -diff` shows nothing; `go.sum` and the requirements are
   unchanged.
3. Run the version-agreement check (a scratch script, not added to the tree):
   it reads the floor and toolchain from `go.mod` and asserts that
   `Makefile`, the CI linter pin and the docs agree, and that no other
   Go-version pin or statement exists. Run it on three planted wrong
   versions in scratch copies: a wrong toolchain, a wrong floor, a stray old
   pin in a workflow.
4. Gates: `make fmt`/`go vet`, `make lint` with a scratch-built v2.15.0 under
   1.27.4 and under `GOTOOLCHAIN=go1.26.11`, the test suite, the floor build,
   vet and tests, `govulncheck` under both toolchains, the shipped
   cross-builds, markdownlint.
5. Stage the record files only.

## Verification

The acceptance criteria are the MADR's Confirmation list, each run and
recorded below with its exit status.

## Rollout and Rollback

Rollout: the coordinator commits the records and the change and pushes `main`;
CI then runs the `validate` and `build` jobs. Rollback: revert the change
commit; the three edits are independent of each other, nothing else moves.

## Execution Record

### Approval (2026-11-17)

The owner approved this pair and its execution through CI ("One short record
pair per repository, all approved now through CI (Recommended)"). Tags and
releases wait for the owner's word.

### Execution (2026-11-17, nothing committed)

* Steps 1 to 3 applied by script with a count assertion each; read back with
  `git diff`. `go.mod:3` `go 1.26.6` to `go 1.26.11`; `go.mod:5` new
  `toolchain go1.27.4`; `Makefile:1` `1.26.6` to `1.26.11`;
  `.github/workflows/ci.yml:20` `@v2.13.1` to `@v2.15.0`. `go mod tidy -diff`
  exit 0; `go.sum` unchanged.
* The Makefile's `MOD_VERSION` is referenced nowhere else: confirmed dead,
  kept and moved.
* Linter baseline: v2.13.1 and v2.15.0, each built into a scratch `GOBIN`
  (v2.15.0 reports "built with go1.27.4"), `make lint` on a scratch copy of
  HEAD under Go 1.26.6: both `0 issues.`, exit 0. So no finding is new.
* Version-agreement check (scratch script): passes on the changed tree;
  planted failures: `go.mod:5 toolchain 1.27.3 != 1.27.4`;
  `go.mod:3 floor 1.26.10 != 1.26.11` with `Makefile:1 MOD_VERSION 1.26.11 !=
  floor 1.26.10`; `.github/workflows/ci.yml:20 lint pin v2.13.1 != v2.15.0`.
* Gates, all exit 0, `GOPROXY=off`: `gofmt -l` empty; `go vet ./...` (1.27.4);
  `make lint` with scratch v2.15.0 under 1.27.4 and under
  `GOTOOLCHAIN=go1.26.11`, `0 issues.` each; floor `go build ./... && go vet
  ./...` under 1.26.11; `go test -race -cover -count=1 -timeout 20m ./...`
  under 1.27.4 (through the heavy-run queue; all packages ok); floor
  `go test -count=1 -timeout 20m ./...` under 1.26.11 (queued; ok);
  `govulncheck ./...` under 1.27.4 and under 1.26.11: "No vulnerabilities
  found.", exit 0 each; cross-builds with `CGO_ENABLED=0` for linux/amd64,
  windows/amd64 and darwin/arm64, exit 0; `markdownlint-cli2` on the README,
  0 issues (the records are excluded by `.markdownlint-cli2.jsonc`).
* Not done: no commit, push, tag or release; no CI run; no floor job added.

## Amendment A1 (2026-11-17): the `mcplib` requirement, after CI went red

Associated decision: Amendment A1 of
[0001-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md](0001-MADR-move-toolchain-to-go-1-27-4-and-floor-to-1-26-11.md).

### Deviation (2026-11-17)

After the push, CI failed on both commits at `go mod download`:
`reading github.com/maccavelli/mcplib/go.mod at revision v1.2.0: unknown
revision v1.2.0`. Pre-existing: the tag left mcplib's source repository, the
proxy answers 404 for it, and the last green run (`ef5ba25`) is older than
that. The owner ruled: "Move the requirement to a mcplib version that
exists, under an amendment (Recommended)". The step "Out of scope: raising
`mcplib`" above is overridden for this one requirement and its indirect
consequence; nothing else changes.

### Steps and results

| Step | File | Change |
| :--- | :--- | :--- |
| A1.1 | `go.mod:14` | `github.com/maccavelli/mcplib v1.2.0` becomes `v1.6.0` |
| A1.2 | `go.mod:19` | `golang.org/x/sys v0.45.0 // indirect` becomes `v0.47.0` |
| A1.3 | `go.mod` | `golang.org/x/tools v0.45.0 // indirect` removed (no longer needed) |
| A1.4 | `go.sum` | `mcplib` v1.6.0 and `x/sys` v0.47.0 lines replace the old ones; `x/tools` v0.45.0 lines become v0.42.0 |

* Scratch first (copy of HEAD): `go get ...@v1.6.0` printed "upgraded
  mcplib v1.2.0 => v1.6.0" and "upgraded x/sys v0.45.0 => v0.47.0"; `go mod
  tidy` and `go mod tidy -diff` exit 0; only `go.mod` and `go.sum` changed.
  Build, vet, `go test -count=1 ./...`, `govulncheck` and `make lint`
  (scratch v2.15.0) exit 0 under 1.27.4 and `GOTOOLCHAIN=go1.26.11`.
* Red before, green after: with `GOMODCACHE` empty, `GOPROXY=https://proxy.golang.org,direct`,
  `GOSUMDB=sum.golang.org`, `GOFLAGS=`, `GOTOOLCHAIN=local`: HEAD fails with
  `go: github.com/maccavelli/mcplib@v1.2.0: reading ...: unknown revision
  v1.2.0`, exit 1; the bumped tree exits 0. `GOPROXY=direct GOSUMDB=off`
  fails the same way on HEAD. Not reproduced: the runner's network.
* Gates again in the tree, all exit 0: `gofmt -l` empty; `go mod tidy -diff`;
  `go vet` (1.27.4); `make lint` with v2.15.0 under 1.27.4 and 1.26.11 (0
  issues each); floor build and vet (1.26.11); `go test -race -cover
  -count=1 -timeout 20m ./...` (1.27.4); floor `go test -count=1 -timeout
  20m ./...` (1.26.11); `govulncheck ./...` under both toolchains ("No
  vulnerabilities found."); `CGO_ENABLED=0` builds for linux/amd64,
  windows/amd64 and darwin/arm64; README markdownlint; `go mod verify`; the
  version-agreement check passes, and its three planted failures still fail.
* Not done: no commit, push or tag. The first CI run on the push is the proof.
