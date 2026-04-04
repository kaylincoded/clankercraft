# Story 6.1: Goreleaser Configuration

Status: done

## Story

As a developer,
I want goreleaser to build binaries for 6 targets on git tag,
so that users can download pre-built binaries for their platform.

## Acceptance Criteria

1. **Given** a git tag is pushed (e.g., `v3.0.0`)
   **When** the GitHub Actions release workflow runs
   **Then** goreleaser produces binaries for linux-amd64, linux-arm64, darwin-amd64, darwin-arm64, windows-amd64, windows-arm64 with checksums

## Tasks / Subtasks

- [x] Task 1: Create `.goreleaser.yaml` (AC: #1)
  - [x] Create `.goreleaser.yaml` in project root
  - [x] Configure `builds` section:
    - Binary name: `clankercraft`
    - Main package: `.` (root)
    - ldflags: `-s -w -X main.version={{.Version}}` (strip debug info + inject version)
    - GOOS targets: `linux`, `darwin`, `windows`
    - GOARCH targets: `amd64`, `arm64`
    - 6 total build targets (3 OS x 2 arch)
  - [x] Configure `archives` section:
    - Format: `tar.gz` for linux/darwin, `zip` for windows
    - Name template: `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}`
  - [x] Configure `checksum` section: SHA256 checksums file
  - [x] Configure `changelog` section: auto-generated from git commits, exclude docs/chore/ci
  - [x] Set `version: 2` (goreleaser v2 config format)
- [x] Task 2: Create GitHub Actions release workflow (AC: #1)
  - [x] Create `.github/workflows/release.yml`
  - [x] Trigger: `on: push: tags: ['v*']`
  - [x] Job steps:
    1. `actions/checkout@v4` with `fetch-depth: 0` (goreleaser needs full git history for changelog)
    2. `actions/setup-go@v5` with Go version from `go.mod`
    3. `goreleaser/goreleaser-action@v6` with `args: release --clean`
  - [x] Set `permissions: contents: write` (needed to create GitHub release)
  - [x] Use `GITHUB_TOKEN` (automatic, no secrets needed)
- [x] Task 3: Update dependabot for goreleaser action (AC: #1)
  - [x] Add `github-actions` ecosystem entry to `.github/dependabot.yml` if not already present (it exists — verify it covers the new workflow)
- [x] Task 4: Validate configuration locally (AC: #1)
  - [x] Run `goreleaser check` to validate `.goreleaser.yaml` syntax
  - [x] Run `goreleaser build --snapshot --clean` to verify builds produce 6 binaries
  - [x] Verify version injection: built binary reports snapshot version (not "dev")
  - [x] All existing tests still pass (`go test ./...`)

## Dev Notes

### Version Injection

`main.go:23` already declares `var version = "dev"`. Goreleaser's ldflags override this at build time:

```yaml
ldflags:
  - -s -w -X main.version={{.Version}}
```

- `-s -w` strips debug symbols (smaller binary)
- `{{.Version}}` is the git tag without the `v` prefix (e.g., tag `v3.0.0` → version `3.0.0`)
- The version is used in: Cobra CLI `--version`, MCP server info, startup log

### Goreleaser v2 Config Format

Use `version: 2` at the top of `.goreleaser.yaml`. Key v2 differences from v1:
- `builds` uses `targets` or `goos`/`goarch` lists (both work)
- `archives` replaces `archive` (plural)
- `format_overrides` for per-OS archive format

### Archive Naming Convention

```yaml
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
```

Produces: `clankercraft_3.0.0_linux_amd64.tar.gz`, `clankercraft_3.0.0_windows_arm64.zip`, etc.

### Existing CI

`.github/workflows/build.yml` is a Node.js CI workflow (npm lint/build/test) from the legacy TypeScript codebase. Leave it as-is — the new `release.yml` is separate and only triggers on tags.

`.github/dependabot.yml` already has `github-actions` ecosystem configured with monthly schedule — the new `goreleaser-action` will be auto-updated.

### What NOT to Build

- **No Go CI workflow yet** — that's Story 6-2 (CI Pipeline: go vet, golangci-lint, go test, go build)
- **No Docker images** — not in scope
- **No homebrew tap** — not in scope
- **No changelog customization beyond basic filtering** — keep it simple

### Local Validation

To validate without publishing a release:

```bash
# Check config syntax
goreleaser check

# Build snapshot (no publish)
goreleaser build --snapshot --clean

# Full dry run (builds + archives + checksums, no publish)
goreleaser release --snapshot --clean
```

The `--snapshot` flag uses `0.0.0-SNAPSHOT-<commit>` as version. The `--clean` flag removes the `dist/` directory first.

### Previous Story Intelligence

- Story 5.6 was the last code story — this is the first infra/config story
- `var version = "dev"` in `main.go:23` is the injection target
- Module path: `github.com/kaylincoded/clankercraft`
- Go version: 1.26.1 (from go.mod)
- Existing `.gitignore` already ignores the `clankercraft` binary and `dist/`

### References

- [Source: main.go#L23] — `var version = "dev"` (ldflags target)
- [Source: main.go#L29] — `Version: version` (Cobra CLI)
- [Source: go.mod] — Module path and Go version
- [Source: .github/workflows/build.yml] — Existing Node.js CI (leave as-is)
- [Source: .github/dependabot.yml] — Existing dependabot config (github-actions already covered)
- [Source: .gitignore] — Already ignores `clankercraft` binary and `dist/`
- [Source: docs/epics.md#Story 6.1] — Original story definition
- [Source: docs/prd.md#FR46] — Distribution requirement
- [External: goreleaser.com/customization] — Goreleaser v2 configuration reference

## Dev Agent Record

### Agent Model Used
Claude Opus 4.6

### Completion Notes List
- `.goreleaser.yaml` v2 config with 6 build targets (linux/darwin/windows x amd64/arm64)
- ldflags inject version via `-X main.version={{.Version}}` + strip debug symbols (`-s -w`)
- Archives: tar.gz default, zip for windows via `format_overrides` with v2 `formats` syntax
- SHA256 checksums in `checksums.txt`
- Changelog auto-generated, excludes docs/chore/ci/test prefixes and merge commits
- GitHub Actions release workflow triggers on `v*` tags, uses goreleaser-action@v6
- `go-version-file: go.mod` in setup-go ensures CI uses same Go version as project
- Dependabot already covers github-actions ecosystem — no changes needed
- Validated locally: `goreleaser check` passes, `goreleaser build --snapshot --clean` produces 6 binaries, version injection confirmed (`2.0.4-SNAPSHOT-aade5f3`)

### File List
- `.goreleaser.yaml` — NEW: Goreleaser v2 configuration (builds, archives, checksums, changelog)
- `.github/workflows/release.yml` — NEW: GitHub Actions release workflow (tag-triggered)
- `docs/stories/sprint-status.yaml` — MODIFIED: Updated 6-1 and epic-6 status
