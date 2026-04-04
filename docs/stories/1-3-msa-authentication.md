# Story 1.3: MSA Authentication

Status: done

## Story

As a user,
I want the bot to authenticate with my Microsoft account,
so that it can join online-mode servers.

## Acceptance Criteria

1. **Given** no cached auth token exists
   **When** the binary starts without `--offline`
   **Then** it initiates MSA device code flow, displays the user code and verification URL on stderr, and waits for the user to authenticate

2. **Given** the user completes the device code flow
   **When** authentication succeeds
   **Then** the bot connects to the server using the obtained MC access token, UUID, and username

3. **Given** a valid cached token exists at `~/.config/clankercraft/tokens/`
   **When** the binary starts
   **Then** it uses the cached token without prompting the device code flow

4. **Given** the cached token is expired but a refresh token exists
   **When** the binary starts
   **Then** it refreshes the token automatically (no user interaction) and connects

5. **Given** MSA authentication fails (invalid account, no MC ownership, network error)
   **When** the auth flow errors
   **Then** an error is logged to stderr and the process exits with non-zero code

6. **Given** `--offline` flag is set
   **When** the binary starts
   **Then** MSA auth is skipped entirely (existing behavior from Story 1.2)

## Tasks / Subtasks

- [x] Task 1: Create `internal/connection/auth.go` — MSA authentication module (AC: #1, #2, #5)
  - [x] Add `go-mc-ms-auth` dependency (`github.com/maxsupermanhd/go-mc-ms-auth`)
  - [x] Create `Authenticate(cfg *config.Config, logger *slog.Logger) (*bot.Auth, error)`
  - [x] Call `msauth.GetMCcredentials(cachePath, clientID)` which handles the full 5-step flow
  - [x] Display device code + verification URL via logger when prompted
  - [x] Return populated `bot.Auth{Name, UUID, AsTk}` on success
  - [x] Return descriptive error on failure (no MC ownership, network error, etc.)
- [x] Task 2: Implement token caching (AC: #3, #4)
  - [x] Cache path: `~/.config/clankercraft/tokens/` (create dir if missing, mode 0700)
  - [x] `go-mc-ms-auth` handles cache file internally via `cachePath` parameter
  - [x] On startup with existing cache: library auto-refreshes if expired
  - [x] Cached tokens skip the device code flow (no user prompt)
- [x] Task 3: Wire auth into connection setup (AC: #2, #6)
  - [x] Modify `setupAuth()` in `mc.go`: if not offline, call auth function to get auth
  - [x] Set `client.Auth.Name`, `client.Auth.UUID`, `client.Auth.AsTk` from result
  - [x] Offline mode unchanged — skip auth entirely (AC #6)
- [x] Task 4: Write tests (all ACs)
  - [x] Test cache directory creation (t.TempDir) with correct permissions
  - [x] Test cache directory creation is idempotent
  - [x] Test `setupAuth` offline path is unchanged (existing tests still pass)
  - [x] Test `setupAuth` online path calls auth and sets all three Auth fields
  - [x] Test `setupAuth` online path propagates auth errors

## Dev Notes

### MSA Authentication Flow (5 steps, handled by go-mc-ms-auth)

```
Step 1: Microsoft Device Code → user_code + verification_uri
        Poll until user authenticates → MSA access_token + refresh_token

Step 2: Xbox Live (XBL) auth → XBL token + user_hash
        POST https://user.auth.xboxlive.com/user/authenticate
        Requires "d=" prefix on MSA token

Step 3: XSTS token exchange → XSTS token
        POST https://xsts.auth.xboxlive.com/xsts/authorize
        RelyingParty: "rp://api.minecraftservices.com/"

Step 4: Minecraft login → MC access_token (24hr lifetime)
        POST https://api.minecraftservices.com/authentication/login_with_xbox
        Identity token format: "XBL3.0 x=<user_hash>;<xsts_token>"

Step 5: Minecraft profile → UUID + username
        GET https://api.minecraftservices.com/minecraft/profile
```

**`go-mc-ms-auth` handles ALL of this.** The library:
- `GetMCcredentials(cachePath, clientID)` — runs the full flow, returns `BotAuth{Name, UUID, AsTk}`
- `CheckRefreshMS(auth, clientID)` — refreshes expired tokens automatically
- Caches tokens to a file at `cachePath`
- Displays device code flow prompts to stdout (we'll need to redirect to stderr)

### Key Library: go-mc-ms-auth

```go
import msauth "github.com/maxsupermanhd/go-mc-ms-auth"

// Full auth flow (first time or expired refresh token):
auth, err := msauth.GetMCcredentials(cachePath, clientID)
// auth.Name = "PlayerName"
// auth.UUID = "undashed-uuid"
// auth.AsTk = "mc-access-token"

// Check if tokens need refresh (subsequent launches):
err := msauth.CheckRefreshMS(&auth, clientID)
```

**BotAuth struct** (from go-mc-ms-auth) matches go-mc's Auth:
```go
type BotAuth struct {
    Name string
    UUID string
    AsTk string
}
```

### Client ID

The `go-mc-ms-auth` library has a default Client ID: `88650e7e-efee-4857-b9a9-cf580a00ef43`. Use this for development. Custom Client IDs require Mojang whitelisting via an Azure App Registration form.

Store client ID as a constant for now. If we need to make it configurable later, add a `--client-id` flag.

### Token Caching Strategy

| Token | Lifetime | Cached By Library |
|---|---|---|
| MSA Refresh Token | ~90 days | Yes (in cache file) |
| MSA Access Token | ~1 hour | Yes |
| MC Access Token | 24 hours | Yes |

- Cache directory: `~/.config/clankercraft/tokens/`
- Cache file created by `go-mc-ms-auth` at the `cachePath` argument
- Directory permissions: 0700 (owner only)
- On first run: device code flow → cache tokens
- On subsequent runs: load cache → refresh if expired → no user interaction
- If refresh token expired (90+ days): fall back to device code flow

### Stderr Redirection

`go-mc-ms-auth` prints the device code prompt to stdout. This is a problem because stdout is reserved for MCP (Story 2.1). Options:
1. The library may accept a writer — check if configurable
2. If not, capture stdout during auth and redirect to stderr
3. Or just log the device code info via slog before calling the library

Preferred: Log the device code ourselves via slog, and suppress/redirect the library's stdout output during auth. The library's `GetMCcredentials` has a callback mechanism — check the API.

### Architecture Compliance

- **`internal/connection/auth.go`** — new file in existing `connection` package. Auth is part of the connection layer.
- **No new package** — auth is connection concern, not a separate boundary
- `setupAuth()` in `mc.go` becomes the integration point: offline → skip, online → call `Authenticate()`
- Config struct already has `Offline` field — no config changes needed

### Security Notes (from PRD NFR18-20)

- MSA tokens cached with restricted file permissions (0700 dir, 0600 file)
- Access tokens MUST NEVER appear in log output — log "authenticated as <username>" not the token
- Refresh tokens MUST NEVER appear in log output
- Cache path under user's config dir — not world-readable

### Previous Story Learnings (Story 1.2)

- `setupAuth(client *bot.Client)` extracted as testable method — extend it, don't replace it
- Offline path sets `Auth.Name` + offline UUID + empty `AsTk` — must remain unchanged
- Online path currently sets only `Auth.Name` with placeholder comment
- `TestOfflineModeAuthSetup` and `TestOnlineModeAuthSetup` exist — extend online test
- go-mc installed from master: v1.20.3-0.20241224032005

### What This Story Does NOT Do

- Custom Client ID configuration (use library default)
- Token encryption at rest (file permissions are sufficient for MVP)
- Multi-account support
- Token revocation on shutdown

### Project Structure Notes

After this story:

```
internal/connection/
├── auth.go          (new — MSA authentication)
├── auth_test.go     (new — auth tests)
├── mc.go            (modified — setupAuth online path)
└── mc_test.go       (modified — extended online auth test)
```

### References

- [Source: docs/prd.md#FR2] — MSA auth with device code flow and token caching
- [Source: docs/prd.md#NFR19] — API key never in CLI args
- [Source: docs/prd.md#NFR20] — MSA tokens cached securely with restricted permissions
- [Source: docs/architecture-decision.md#Authentication] — MSA via go-mc, device code, token caching
- [Source: docs/epics.md#Story 1.3] — Original story definition
- [Source: docs/stories/1-2-minecraft-connection-via-go-mc.md] — Previous story learnings
- [External: github.com/maxsupermanhd/go-mc-ms-auth] — MSA auth library for go-mc
- [External: wiki.vg/Microsoft_Authentication_Scheme] — Full auth flow documentation

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Completion Notes List

- go-mc-ms-auth installed: v0.0.0-20230820124717-22f4d907eac4
- auth.go: Authenticate() wraps msauth.GetMCcredentials() with cache dir setup, returns *bot.Auth
- ensureCacheDir() creates ~/.config/clankercraft/tokens/ with 0700 permissions
- Library handles full MSA flow: device code → XBL → XSTS → MC login → profile, plus token caching and refresh
- Library uses log.Print (stderr), NOT stdout — no stdout redirection needed (story spec concern resolved)
- setupAuth() now returns error, calls injected authFn for online mode
- AuthFunc type + authFn field on Connection enables test injection without interfaces
- Offline path unchanged — sets Auth.Name + offline UUID, no auth call
- 28 total tests across all packages, all passing (4 new in auth_test.go, 1 new + 1 updated in mc_test.go)

### Code Review Fixes Applied

- **M1 — ensureCacheDir() doesn't enforce permissions on existing directories**: Added `os.Chmod(dir, 0700)` after `MkdirAll` to enforce permissions regardless of prior state. Added `TestEnsureCacheDirFixesPermissions` test.
- **L1 — Redundant HOME env var cleanup**: Removed manual `origHome` + `defer os.Setenv` — `t.Setenv` handles restoration automatically.
- **L2 — Device code flow uninterruptible by SIGINT**: Skipped — library limitation, out of scope.

### File List

- `internal/connection/auth.go` (new — MSA authentication)
- `internal/connection/auth_test.go` (new — auth + cache dir tests)
- `internal/connection/mc.go` (modified — setupAuth returns error, uses injected authFn)
- `internal/connection/mc_test.go` (modified — updated for setupAuth error return, added online auth tests)
- `go.mod` (modified — added go-mc-ms-auth dependency)
- `go.sum` (modified)
