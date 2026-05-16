# Technical Concerns

**Analysis Date:** 2026-05-16

## High Priority

### Non-atomic State Writes
**File:** `pkg/state/local.go`
**Risk:** If nim crashes mid-write, `~/.nim/state.json` is left partially written and corrupted. Next run will fail to parse state and may re-apply all resources.
**Fix:** Write to a temp file, then `os.Rename()` (atomic on POSIX).

### AISkillProvider.getInstalledSources Always Returns Empty
**File:** `pkg/providers/aiskill.go`
**Risk:** Skills are re-installed on every `nim apply` regardless of whether they're already present. No idempotency.
**Fix:** Implement actual detection of installed AI skill sources.

### S3 Credentials Stored Plaintext
**File:** `~/.nim/config.yaml` (runtime concern), `pkg/state/s3.go`
**Risk:** `state.s3.access_key` and `state.s3.secret_key` stored as plaintext in config file. Any process that can read the home directory gets S3 credentials.
**Fix:** Support env var fallback (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`) and document not to put credentials in config.

## Medium Priority

### Duplicate S3Config Struct
**Files:** `pkg/config/config.go` (defines `S3Config`), `pkg/state/s3.go` (defines its own `s3Config`)
**Risk:** Two struct definitions with overlapping fields require manual field mapping. Adding a field to one doesn't automatically update the other.
**Fix:** Consolidate into a single shared type.

### State Path Hardcoded Three Times
**File:** `cmd/state.go`
**Risk:** Default state path (`~/.nim/state.json`) is hardcoded in 3 places instead of being read from the config/backend factory. Changing the default requires updating all 3 sites.
**Fix:** Use the config-resolved backend path consistently.

### ensureProvidersRegistered() Duplication
**File:** `cmd/state.go`
**Risk:** `ensureProvidersRegistered()` in `cmd/state.go` registers providers with an empty dotfiles root, diverging from engine registration. State subcommand sees a different provider set than plan/apply.
**Fix:** Share provider registration with the engine initialization path.

### Deprecated Legacy State Methods
**File:** `pkg/state/state.go`
**Risk:** `GetResource`, `SetResource`, `RemoveResource` methods still present with incorrect semantics. Callers may use them expecting correct behavior.
**Fix:** Remove deprecated methods; callers should use the batch `Save`/`Load` API.

### brew --prefix Fallback Hardcoded
**File:** `pkg/providers/cask.go` (or `brew.go`)
**Risk:** Fallback hardcoded to `/opt/homebrew` breaks Intel Mac cask version detection (`/usr/local/Homebrew`). Intel Mac users see incorrect version reporting.
**Fix:** Run `brew --prefix` and handle both architectures, or use `brew info --json` consistently.

## Low Priority

### No Tests for Key Providers
**Files:** `pkg/providers/go.go`, `pkg/providers/npm.go`, `pkg/providers/aiskill.go`, `pkg/state/s3.go`
**Risk:** Core provider logic has zero unit test coverage. Regressions won't be caught by CI.
**Fix:** Add table-driven unit tests; mock external CLI calls with an exec abstraction.

### context.Background() in Tree Rendering
**File:** `pkg/diff/tree.go` (or `pkg/ui/`)
**Risk:** Tree rendering ignores context cancellation during `nim plan` display. Ctrl+C during plan output may not interrupt rendering.
**Fix:** Thread ctx through the treeview render call.

### Static fmt.Sprintf with No Format Args
**File:** `pkg/engine/apply.go:246`
**Risk:** `fmt.Sprintf("dependency failed")` — pointless allocation; `go vet` may flag this in future versions.
**Fix:** Replace with a bare string `"dependency failed"`.

---

*Concerns analysis: 2026-05-16*
