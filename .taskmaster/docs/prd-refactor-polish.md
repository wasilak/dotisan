# Dotisan: Senior-Level Refactor & Charm Ecosystem Polish

## Overview

Dotisan is a dotfiles/package manager CLI built incrementally via agentic AI work. It is functional but carries significant technical debt characteristic of AI-generated codebases: copy-paste implementations, hardcoded strings, business logic entangled with UI, and debug artifacts left in production code. This PRD defines a systematic refactoring and polish effort to make the codebase maintainable, idiomatic Go, and visually excellent using the full Charmbracelet ecosystem.

## Goals

1. Eliminate code duplication and establish proper abstractions across all providers
2. Remove debug artifacts and fix silent error handling
3. Decompose the monolithic engine.go into focused, single-responsibility files
4. Replace custom terminal hacks with proper charmbracelet library components
5. Add visual richness and polish using bubbles, huh, and lipgloss while keeping JSON output clean
6. Make the codebase approachable for future development

## Non-Goals

- Changing the YAML resource format or CLI flag interface (backward compat required)
- Adding new provider types or resource kinds
- Removing bubbletea entirely (bubbles, which we keep, requires it; and we're adding huh which can use it)
- Touching S3 state backend logic

---

## Phase 1: Code Quality Refactor

### R1 — Extract Provider Base Reconciliation Pattern

**Problem:** The four package providers (brew, npm, go, cargo) each implement Reconcile() and compareGroupItems() with 90–95% identical code — verbatim copy-paste. This means bug fixes and improvements must be applied 4 times, and always drift out of sync.

**Scope:**
- `pkg/providers/npm.go` (333 lines)
- `pkg/providers/cargo.go` (333 lines, ~95% identical to npm)
- `pkg/providers/go.go` (428 lines)
- `pkg/providers/brew.go` (620 lines, more complex due to cask/tap handling)
- New file: `pkg/provider/reconcile.go`

**Requirements:**
- Extract shared reconcile scaffolding into `pkg/provider/reconcile.go`:
  - `IndexStateByGroup(state ResourceState, kind string) map[string][]provider.StateItem` — shared indexing
  - `CompareGroupItems(desired, current []string, installed map[string]bool) GroupDiff` — addition/removal/modification detection
  - `BaseReconcile(desired []ResourceGroup, stateIndex map[string][]StateItem, getInstalled func() (map[string]bool, error)) ([]GroupPlan, error)` — shared skeleton
- Each provider implements only the provider-specific parts: `getInstalledPackages()`, `apply*()`
- Brew provider may need a thin wrapper due to cask/tap complexity, but should reuse CompareGroupItems
- All existing tests must continue to pass after extraction
- No behavioral changes — pure refactor

**Verification:**
- `go build ./...`
- `go test ./...`
- `dotisan plan` produces identical output before/after

---

### R2 — Define Constants: Eliminate Hardcoded Strings

**Problem:** Resource kind strings appear 40+ times as raw string literals throughout the codebase. The "default" namespace string appears 20+ times. Status strings like "present" and reason strings are scattered. `fmt.Sprintf("%s/%s/%s", kind, group, item)` is repeated 10+ times.

**Scope:**
- New file: `pkg/resource/constants.go`
- New file: `pkg/engine/constants.go`
- Update all files referencing these strings: `pkg/engine/engine.go`, `pkg/engine/apply.go`, `cmd/state.go`, `pkg/providers/*.go`, `pkg/resource/unmarshal.go`

**Requirements:**
- `pkg/resource/constants.go`:
  ```go
  const (
      KindHomeBrewPackages    = "HomeBrewPackages"
      KindNpmPackages     = "NpmPackages"
      KindGoPackages      = "GoPackages"
      KindCargoPackages   = "CargoPackages"
      KindManagedFile     = "ManagedFile"
      // KindManagedDirectory removed: ManagedDirectory no longer exists
  )
  
  func ResourceID(kind, group, item string) string {
      return fmt.Sprintf("%s/%s/%s", kind, group, item)
  }
  ```
- `pkg/engine/constants.go`:
  ```go
  const (
      DefaultNamespace = "default"
      StatusPresent    = "present"
      StatusAbsent     = "absent"
  )
  ```
- Replace every hardcoded occurrence with constants using codebase-memory-mcp to find all call sites
- No behavioral changes

**Verification:**
- `go build ./...`
- `go test ./...`
- grep for raw "BrewPackages", "NpmPackages" strings confirms zero remaining in non-constant code

---

### R3 — Decompose engine.go (979 Lines)

**Problem:** `pkg/engine/engine.go` is 979 lines containing: Plan() logic (161 lines), ApplyWithProgress() (248 lines with embedded TUI), cleanup detection, resource loading, target filtering (118 lines), and the full bubbletea progress model. `pkg/engine/apply.go` also exists with overlapping apply logic, creating confusion.

**Scope:**
- `pkg/engine/engine.go` (source, 979 lines)
- `pkg/engine/apply.go` (existing, 215 lines — overlaps)
- New file: `pkg/engine/plan.go` — Plan() logic and filterPlanByTargets()
- New file: `pkg/engine/cleanup.go` — cleanup/orphan detection logic
- New file: `pkg/engine/progress.go` — applyProgressModel and all bubbletea TUI code
- Keep `pkg/engine/engine.go` as thin coordinator with Engine struct, NewEngine(), and public method routing only
- Reconcile the two Apply implementations: `engine.go:ApplyWithProgress()` and `apply.go:Apply()` — there should be one clear path

**Requirements:**
- `pkg/engine/plan.go`: Plan(), loadResources(), groupResources(), filterPlanByTargets(), target matching logic
- `pkg/engine/cleanup.go`: cleanup detection (orphaned state items no longer in config)
- `pkg/engine/progress.go`: applyProgressModel struct, Init/Update/View, applyProgressMsg/applyCompleteMsg/applyTickMsg, tickCmd()
- `pkg/engine/apply.go`: keep as the single Apply entrypoint; remove duplication with engine.go
- `pkg/engine/engine.go`: Engine struct, NewEngine(), public facade methods only — no business logic
- All method signatures remain identical (public API unchanged)
- Use Serena for symbol-level moves to avoid manual copy-paste errors

**Verification:**
- `go build ./...`
- `go test ./...`
- `dotisan plan`, `dotisan apply --confirm` work identically

---

### R4 — Fix Error Handling

**Problem:** Multiple places silently swallow errors, making debugging impossible:
- `pkg/providers/go.go`, `npm.go`, `cargo.go`: `getInstalledPackages()` returns empty map on error with no logging
- `cmd/root.go`: `config.LoadConfigFromDefaultPath()` error ignored (`cfg, _ :=`)
- `cmd/state.go:121`: config loaded but nil not checked before access
- `cmd/doctor.go:56`: `provider.CheckAvailable()` error result ignored

**Scope:**
- `pkg/providers/go.go`, `pkg/providers/npm.go`, `pkg/providers/cargo.go`
- `cmd/root.go`
- `cmd/state.go`
- `cmd/doctor.go`

**Requirements:**
- `getInstalledPackages()` errors: use `slog.Warn("failed to get installed packages", "provider", name, "error", err)` and return the error up the stack (don't silently continue with empty map)
- `cmd/root.go`: handle config load error explicitly — if config missing, use defaults and log at debug; if parse error, log warn
- `cmd/state.go`: add nil check for cfg after load; return early with user-friendly error message if nil
- `cmd/doctor.go`: capture and display `CheckAvailable()` errors per provider
- Use `slog` (already initialized globally in cmd/root.go) consistently — no fmt.Fprintf(os.Stderr) for errors
- No new dependencies

**Verification:**
- `go build ./...`
- `go test ./...`
- `dotisan doctor` with a missing provider shows error instead of silent skip

---

### R5 — Remove Debug Artifacts

**Problem:** AI-generated debug log dumps remain in production code:
- `pkg/providers/brew.go`: 4 DEBUG dump locations (lines ~164-166, ~296-310, ~347-348, ~391-396)
- `pkg/resource/brew.go`: DEBUG spec count dump
- `pkg/resource/loader.go`: DEBUG rendered resource print

**Scope:**
- `pkg/providers/brew.go`
- `pkg/resource/brew.go`
- `pkg/resource/loader.go`

**Requirements:**
- Remove all DEBUG comment blocks and their associated print/log statements
- Surgical removal only — do NOT refactor surrounding code
- If any debug output is genuinely useful for troubleshooting, convert to `slog.Debug(...)` calls with appropriate fields
- No behavioral change to production behavior

**Verification:**
- `go build ./...`
- `go test ./...`
- grep for "DEBUG" in .go files returns zero results

---

## Phase 2: Charm Ecosystem Upgrade

### C1 — Add huh + Replace Confirmation Prompts

**Problem:** `pkg/style/confirm.go` implements a custom raw terminal reader using `golang.org/x/term` with `ReadSingleKey()`. This works but is fragile, non-accessible, and bypasses the charm ecosystem we already use. `huh` provides a polished, accessible `Confirm` component that works standalone (no bubbletea program needed for simple cases).

**Scope:**
- `pkg/style/confirm.go` (replace/rewrite)
- `pkg/style/styles.go` (remove ConfirmBox helper, or keep as lipgloss-only for non-huh paths)
- `cmd/apply.go` (uses confirmation at lines ~120-139)
- `cmd/state.go` (uses confirmation at lines ~253-279)
- `go.mod` / `go.sum` (add huh dependency)

**Requirements:**
- Add `github.com/charmbracelet/huh` as a direct dependency
- Create `pkg/style/confirm.go` with:
  ```go
  func Confirm(title, affirmativeLabel, negativeLabel string) (bool, error) {
      var confirmed bool
      return confirmed, huh.NewConfirm().
          Title(title).
          Affirmative(affirmativeLabel).
          Negative(negativeLabel).
          Value(&confirmed).
          Run()
  }
  ```
- Replace all call sites in cmd/apply.go and cmd/state.go to use this new function
- Remove `ReadSingleKey()` and the old `ConfirmBox()` style helper if no longer used
- The huh confirm prompt should use the application's existing color theme (huh supports custom themes via `huh.ThemeCharm()` or custom theme)
- Handle Ctrl+C / ErrUserAborted gracefully — treat as "no"

**Verification:**
- `go build ./...`
- `go test ./...`
- `dotisan apply` shows huh confirm prompt
- `dotisan state remove Kind/Group` shows huh confirm prompt
- Ctrl+C on prompt cancels cleanly

---

### C2 — Add Bubbles Spinner for Long-Running Operations

**Problem:** `dotisan doctor` runs provider checks sequentially with no feedback. `dotisan init` creates files with no progress indicator. Users see a blank terminal during these operations.

**Scope:**
- `cmd/doctor.go`
- `cmd/init.go`
- `cmd/state.go` (state pull/push if they have any latency)
- New helper: `pkg/style/spinner.go`

**Requirements:**
- Create `pkg/style/spinner.go` with a simple `RunWithSpinner(label string, fn func() error) error` helper that:
  - Starts a bubbles spinner in a bubbletea program
  - Runs `fn` in a goroutine
  - Shows spinner + label while running
  - Stops and cleans up on completion or error
- `cmd/doctor.go`: wrap each provider check in `RunWithSpinner("Checking <provider>...", ...)`
- `cmd/init.go`: show spinner during template copy / directory creation steps
- Spinner style: use `spinner.MiniDot` or `spinner.Dot` with lipgloss styling matching the app theme
- Do NOT show spinner when `--output json` or when stdout is not a TTY

**Verification:**
- `go build ./...`
- `dotisan doctor` shows animated spinner per provider
- `dotisan init` shows spinner during setup

---

### C3 — Enhance Apply Progress Display

**Problem:** The current `applyProgressModel` in engine.go shows a minimal progress bar with percentage and 3 recent items. It lacks: task names during execution, elapsed time, clear success/error totals, and visual separation.

**Scope:**
- `pkg/engine/progress.go` (after R3 extraction)
- `pkg/engine/engine.go` (ApplyWithProgress orchestration)

**Requirements:**
- Enhanced progress View() to show:
  - Large styled header: "Applying changes..." with lipgloss styling
  - Progress bar with gradient (keep existing)
  - Current task label: "→ Installing brew formula: ripgrep"
  - Elapsed time counter
  - Rolling log of last 5 completed items with ✓/✗ icons and lipgloss coloring
  - Footer summary: "3 succeeded · 1 failed" using style.Success/style.Error colors
- Use elapsed time via `time.Since(startTime)` tracked in model
- Use lipgloss for the "Applying changes..." header (bold, themed color)
- Recent items list: show max 5 items, scroll up as new ones arrive
- On completion: clear progress bar, show final summary box using existing `style.SuccessBox` / `style.ErrorBox`
- Keep existing message types (applyProgressMsg, applyCompleteMsg) — extend as needed

**Verification:**
- `go build ./...`
- `dotisan apply` with real changes shows enhanced progress
- Completion shows summary box with counts

---

### C4 — Centralize Key Bindings with bubbles/key

**Problem:** Key handling is ad-hoc — raw byte reads via the custom confirm.go, hardcoded in the bubbletea Update() methods. As we add huh (C1), the raw key read is eliminated for confirmations. But the bubbletea progress model still handles Ctrl+C inline with a raw switch case.

**Scope:**
- `pkg/engine/progress.go` (after R3)
- `pkg/style/keys.go` (new)

**Requirements:**
- Create `pkg/style/keys.go` defining shared key bindings:
  ```go
  var DefaultKeys = struct {
      Quit key.Binding
      Confirm key.Binding
      Cancel key.Binding
  }{
      Quit:    key.NewBinding(key.WithKeys("ctrl+c", "q"), key.WithHelp("ctrl+c", "quit")),
      Confirm: key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y", "confirm")),
      Cancel:  key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n", "cancel")),
  }
  ```
- Replace inline `case "ctrl+c":` handling in progress model Update() with `key.Matches(msg, style.DefaultKeys.Quit)`
- This ensures consistent key handling if more interactive elements are added

**Verification:**
- `go build ./...`
- Ctrl+C during apply still cancels
- No behavioral regression

---

## Phase 3: Visual Polish

### V1 — Plan Output Enhancement

**Problem:** The plan output (plain and tree formats) is functional but visually plain. The tree format could leverage lipgloss borders, styled section headers, and richer diff indicators. The summary line at the bottom is minimal.

**Scope:**
- `pkg/diff/plan.go` (228 lines)
- `pkg/diff/tree.go` (238 lines)
- `pkg/style/styles.go` (add new styles as needed)

**Requirements:**
- **Plain format** (`pkg/diff/plan.go`):
  - Add styled section dividers between provider groups (e.g., "── HomeBrewPackages ──" in dim color)
  - Add colored summary box at bottom: use existing `style.SuccessBox`/`style.WarningBox` with counts
  - Show drift items with a different icon (⚠ amber) vs additions (✚ green) vs removals (✖ red)
  - Unchanged: keep JSON output format completely clean (no lipgloss in json path)
- **Tree format** (`pkg/diff/tree.go`):
  - Add lipgloss-styled root node showing provider name in bold with item count
  - Show addition/removal/modification counts inline with the group name: "mygroup (2 add, 1 remove)"
  - Color-code leaf nodes by change type
- Add a `NoPendingChanges()` styled message for when plan shows 0 changes (currently just shows nothing)

**Verification:**
- `go build ./...`
- `dotisan plan` shows enhanced plain output
- `dotisan plan -o tree` shows enhanced tree
- `dotisan plan -o json` produces identical clean JSON as before

---

### V2 — Doctor Output Polish

**Problem:** `dotisan doctor` outputs plain text provider checks with no visual structure. Results are hard to scan quickly.

**Scope:**
- `cmd/doctor.go` (282 lines)

**Requirements:**
- Group checks into sections with lipgloss styled headers:
  - "System Prerequisites" — git, curl, etc.
  - "Providers" — brew, npm, go, cargo
  - "Configuration" — config file, state backend
- Each check shows: icon (✓ green / ✗ red / ⚠ yellow) + check name + result
- Use `style.Success`, `style.Error`, `style.Warning` for the icons
- Final summary: "All checks passed" in SuccessBox, or "N checks failed" in ErrorBox
- With C2 (spinner), each section header shows spinner while checks run, then switches to final state
- Validate flag (`--validate`) results shown in a separate "Resource Validation" section

**Verification:**
- `go build ./...`
- `dotisan doctor` shows structured, colorful output
- `dotisan doctor --validate` shows validation results in dedicated section

---

### V3 — State List, Welcome Banner & General Polish

**Problem:** Various output surfaces are inconsistent or plain:
- `dotisan state list` table is functional but minimal
- Root command shows a plain welcome message
- Box styles are consistent in code but could use refinement

**Scope:**
- `cmd/state.go` (state list table, lines ~399-431)
- `cmd/root.go` (welcome/usage output)
- `pkg/style/styles.go` (add/refine shared styles)

**Requirements:**
- **State list table** (`cmd/state.go`):
  - Color-code STATUS column: "managed" in green, "drifted" in yellow, "orphaned" in red
  - Add row striping using RowSuccess/RowWarning variants
  - Bold the KIND column for visual hierarchy
  - Add a footer line showing total count: "Showing N managed resources"
- **Welcome banner** (`cmd/root.go` or `cmd/root.go` help template):
  - Styled app name with lipgloss: bold, themed color
  - One-line tagline in dim style
  - This only shows in interactive terminal mode (not piped/json)
- **Consistency pass** (`pkg/style/styles.go`):
  - Audit all box styles — ensure SuccessBox, ErrorBox, WarningBox, InfoBox have consistent padding/border
  - Add a `SectionHeader(title string) string` helper for consistent section dividers used across V1/V2
- JSON output paths: add a `IsTerminal() bool` helper in pkg/style to guard color output; ensure all styled output checks this

**Verification:**
- `go build ./...`
- `dotisan state list` shows colorized table with footer
- `dotisan state list -o json` produces clean JSON
- `dotisan` (bare) shows styled banner

---

## Implementation Notes for All Tasks

- **Use codebase-memory-mcp extensively**: Before every edit, run `trace_call_path` to understand inbound references. Run `search_code` to validate assumptions. Run `detect_changes` after edits to see blast radius.
- **Use Serena for all code navigation and editing**: `get_symbols_overview`, `find_symbol`, `find_referencing_symbols`, `replace_symbol_body`, `insert_after_symbol`. Do NOT use Read/Edit/Grep on code files when Serena can do it better.
- **Test after each task**: `go build ./...` and `go test ./...` must pass before moving to next task
- **JSON output guard**: every styling addition must check that `--output json` paths remain unaffected
- **No new CLI flags or config keys** unless explicitly stated
- **macOS only**: no Linux/Windows compatibility concerns for this project

## Task Execution Order

Dependencies:
- R3 must come after R1 (provider extraction first, then engine decomp)
- R2 can run in parallel with R1
- R5 can run anytime (pure removal, no dependencies)
- C1 depends on nothing in Phase 1 (can overlap with R4/R5)
- C2/C3/C4 should come after C1 (consistent approach to interactive elements)
- V1/V2/V3 should come after Phase 2 (use new style helpers)

Recommended order: R5 → R2 → R4 → R1 → R3 → C1 → C2 → C4 → C3 → V1 → V2 → V3
