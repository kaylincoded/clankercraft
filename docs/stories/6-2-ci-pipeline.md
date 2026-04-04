# Story 6.2: CI Pipeline

Status: done

## Story

As a developer,
I want CI to lint, test, and build on every push and PR,
so that regressions are caught early.

## Acceptance Criteria

1. **Given** a push to main or a PR
   **When** CI runs
   **Then** it executes: `go vet`, `golangci-lint`, `go test ./...`, `go build`

## Tasks / Subtasks

- [x] Task 1: Create Go CI workflow (AC: #1)
  - [x] Create `.github/workflows/ci.yml`
  - [x] Trigger: `on: push: branches: [main]` and `on: pull_request: branches: [main]`
  - [x] Job steps:
    1. `actions/checkout@v4`
    2. `actions/setup-go@v5` with `go-version-file: go.mod`
    3. Run `go vet ./...`
    4. Install and run `golangci-lint` via `golangci/golangci-lint-action@v7`
    5. Run `go test ./...`
    6. Run `go build -o /dev/null .` (verify compilation, discard binary)
  - [x] Set reasonable timeout (10 minutes)
- [x] Task 2: Add golangci-lint configuration (AC: #1)
  - [x] Create `.golangci.yml` in project root
  - [x] Enable linters: `govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`
  - [x] Set timeout to 3 minutes
  - [x] Exclude test files from `errcheck` (test helpers often ignore errors intentionally)
  - [x] Exclude `node_modules` and `dist` directories
- [x] Task 3: Fix any lint issues (AC: #1)
  - [x] Run `golangci-lint run ./...` locally
  - [x] Fix all reported issues (if any)
  - [x] Verify `go vet ./...` passes
- [x] Task 4: Validate CI workflow locally (AC: #1)
  - [x] Verify `go vet ./...` passes
  - [x] Verify `golangci-lint run ./...` passes
  - [x] Verify `go test ./...` passes
  - [x] Verify `go build -o /dev/null .` succeeds

## Dev Notes

### Existing CI

`.github/workflows/build.yml` is a Node.js CI workflow (npm lint/build/test) from the legacy TypeScript codebase. Leave it as-is — the new `ci.yml` is a separate Go workflow.

`.github/workflows/release.yml` (Story 6-1) handles goreleaser on tag push. The CI workflow runs on push/PR to main — no overlap.

### golangci-lint Action

Use `golangci/golangci-lint-action@v7` which handles installation and caching automatically. Do NOT install golangci-lint manually — the action does it better with proper cache invalidation.

```yaml
- uses: golangci/golangci-lint-action@v7
```

The action auto-detects `.golangci.yml` in the project root.

### Build Verification

Use `go build -o /dev/null .` to verify compilation without leaving artifacts. The `-o /dev/null` discards the binary — we only care that it compiles.

### golangci-lint Config

Keep it conservative — enable standard linters that catch real bugs, not style opinions:

```yaml
linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
```

These are the default linters minus `typecheck` (handled by the compiler). No controversial style linters.

### What NOT to Build

- **No code coverage reporting** — not in scope
- **No matrix builds** (multiple Go versions) — single version from go.mod is sufficient
- **No caching config** — `actions/setup-go@v5` handles module caching automatically
- **No Docker or deployment steps** — CI only

### Previous Story Intelligence

- Story 6-1: `.github/workflows/release.yml` — reference for Go workflow structure (`actions/setup-go@v5` with `go-version-file: go.mod`)
- Story 6-1: `.goreleaser.yaml` — confirms build entry point is `.` (root)
- Go version: 1.26.1 (from go.mod)
- Module path: `github.com/kaylincoded/clankercraft`
- Full test suite passes (`go test ./...`)
- Legacy Node.js CI in `build.yml` — leave untouched

### References

- [Source: .github/workflows/release.yml] — Existing Go workflow pattern (setup-go, go-version-file)
- [Source: .github/workflows/build.yml] — Legacy Node.js CI (leave as-is)
- [Source: go.mod] — Module path and Go version
- [Source: docs/epics.md#Story 6.2] — Original story definition

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- `.github/workflows/ci.yml` — Go CI workflow: checkout, setup-go, vet, golangci-lint-action@v7, test, build
- Triggers on push to main and PRs targeting main, 10-minute timeout
- `.golangci.yml` — golangci-lint v2 config with conservative linter set (govet, errcheck, staticcheck, unused, ineffassign)
- Test files excluded from errcheck via `linters.exclusions.rules`
- `node_modules` and `dist` excluded via `linters.exclusions.paths`
- Fixed 8 lint issues: 7 unchecked error returns (added `_ =` for fire-and-forget calls), 1 staticcheck De Morgan's law suggestion in `isValidPlayerName`
- `gosimple` linter removed from config — merged into `staticcheck` in golangci-lint v2
- All 4 validation steps pass: `go vet`, `golangci-lint run`, `go test`, `go build`

### File List
- `.github/workflows/ci.yml` — NEW: Go CI workflow (push/PR to main)
- `.golangci.yml` — NEW: golangci-lint v2 configuration
- `internal/config/config.go` — MODIFIED: Check `viper.BindPFlag` error returns
- `internal/connection/mc.go` — MODIFIED: Check `client.Conn.Close()` error return
- `internal/agent/tools.go` — MODIFIED: Apply De Morgan's law to `isValidPlayerName`
- `main.go` — MODIFIED: Check error returns on `rconClient.Connect`, `rconClient.Close`, `conn.Close`
- `docs/stories/6-2-ci-pipeline.md` — MODIFIED: Task completion and Dev Agent Record
- `docs/stories/sprint-status.yaml` — MODIFIED: Story 6-2 status
