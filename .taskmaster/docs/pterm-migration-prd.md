# PRD: Migrate from charm* Libraries to pterm

## Background

dotisan currently depends on 4 charm* libraries (`lipgloss`, `huh`, `bubbletea`, `bubbles`),
two of which have zero actual usage and exist only in go.mod. The active charm* usage is
fragmented: lipgloss for styling, huh for confirm dialogs, a custom ASCII spinner (not even
using the charm spinner), and a custom table renderer.

The goal of this migration is to replace the entire charm* stack with **pterm** — a single,
cohesive static-output CLI library that covers every component currently in use and unblocks
future features (progress bar for apply, task 9 in the roadmap) without requiring bubbletea.

## Library Evaluation Summary

Five alternatives were evaluated:

- **tview** — full-screen ncurses TUI requiring an Application event loop. Wrong paradigm
  for a CLI that prints static lines to stdout like `terraform plan`. Rejected.
- **tcell** — low-level terminal cell API, addresses individual (row,col) positions. Used
  as backend by tview. No components. Rejected.
- **termenv** — styling-only library (color, bold, ANSI escape). Covers ~30% of our needs.
  No table, spinner, tree, or interactive prompts. Lipgloss is conceptually built on top of
  termenv. Rejected as primary replacement.
- **pterm** — pure static-output CLI library. No event loop. Covers every component we use
  today and everything planned. **Selected.**

## Current charm* Inventory

| Package | Status | What it does |
|---|---|---|
| `charm.land/lipgloss/v2` | Active | Colors, bold, borders, padding in 5 files |
| `charm.land/lipgloss/v2/tree` | Active | Tree rendering (`pkg/diff/tree.go`) |
| `charm.land/huh/v2` | Active | Confirm dialogs (`cmd/apply.go`, `cmd/state.go`) |
| `charm.land/bubbletea/v2` | Unused | go.mod only — zero import usage |
| `charm.land/bubbles/v2` | Unused | go.mod only — zero import usage |

## pterm Component Mapping

| Need | Current | pterm |
|---|---|---|
| Text colors, bold | `lipgloss.NewStyle().Foreground()` | `pterm.FgGreen.Sprint()`, `pterm.NewStyle()` |
| Borders / boxes | `lipgloss.RoundedBorder()` | `pterm.DefaultBox` |
| Table | Custom `pkg/ui/table.go` + lipgloss cells | `pterm.DefaultTable.WithData()` — auto-sizes columns |
| Tree | `lipgloss/tree` (`tree.Root()`, `tree.New()`) | `pterm.DefaultTree.WithRoot()` |
| Spinner | Custom ASCII frames `pkg/style/spinner.go` | `pterm.DefaultSpinner.Start()/.Success()/.Fail()` |
| Confirm dialog | `huh.NewConfirm().Run()` | `pterm.DefaultInteractiveConfirm.Show()` |
| Progress bar | bubbletea (planned, unused) | `pterm.DefaultProgressbar` |

**Table note:** pterm `TableData` is `[][]string`. Per-cell coloring is done by
pre-coloring strings with `pterm.FgGreen.Sprint("+")` before inserting into TableData.
pterm strips ANSI escape codes for width calculation, so column alignment stays correct.
Content-driven column widths (our current custom logic) become unnecessary — pterm
auto-sizes all columns to content by default.

## Requirements

1. All existing CLI output must remain functionally identical after each phase:
   plan, apply, state list, state remove, doctor, init
2. `go build ./...` and `go test ./...` must pass after each phase independently
3. Non-TTY fallback must be preserved — no spinner/confirm when stdout is piped
4. JSON output paths (`-o json`) must be completely unaffected
5. Phase 1 must be deliverable without Phase 2 (no partial broken state)

## Out of Scope

- Changing command behavior, argument formats, or output structure
- Adding new CLI features beyond the library swap itself
- Moving to interactive TUI mode (that would require tview/tcell)

---

## Phase 1: Drop Dead Weight + Replace Interactive Components

**Goal:** Remove the 3 heaviest/unused charm dependencies. Replace huh confirm and
custom spinner with pterm equivalents.

**Risk:** Low. No changes to static table/tree output. Only interactive prompt appearance
and spinner animation change.

### Task 1.1 — Add pterm, remove bubbletea + bubbles + huh

Update `go.mod`:
- `go get github.com/pterm/pterm`
- `go get -u` then remove `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/huh/v2`
- Run `go mod tidy`

Verify `go build ./...` still compiles (blank imports `_ "charm.land/lipgloss/v2"` in
`cmd/plan.go` and `cmd/apply.go` may need cleanup).

### Task 1.2 — Replace custom spinner with pterm.DefaultSpinner

File: `pkg/style/spinner.go`

Current implementation: custom ticker with ASCII frames (`|`, `/`, `-`, `\`), goroutine-based.
Replace with `pterm.DefaultSpinner`. The public API `RunWithSpinner(ctx, msg, fn)` must be
preserved since it is called from `cmd/plan.go`, `cmd/apply.go`, `cmd/state.go`, `cmd/doctor.go`.

The pterm spinner respects TTY automatically — it degrades gracefully when stdout is not a
terminal. Remove the manual `term.IsTerminal` check inside the current implementation
(pterm handles this internally).

New signature stays the same:
```go
func RunWithSpinner(ctx context.Context, msg string, fn func(ctx context.Context) error) error
```

pterm spinner pattern:
```go
spinner, _ := pterm.DefaultSpinner.Start(msg)
err := fn(ctx)
if err != nil {
    spinner.Fail(msg)
} else {
    spinner.Success(msg)
}
return err
```

### Task 1.3 — Replace huh confirm in cmd/apply.go

File: `cmd/apply.go`

Find the `huh.NewConfirm()` block (currently around line 240). Replace with:
```go
result, _ := pterm.DefaultInteractiveConfirm.Show(title)
confirm = result
```

Remove the manual TTY detection branch — pterm's interactive confirm handles TTY fallback
internally. Keep the `if !isTTY` branch only if pterm does not degrade gracefully on
non-TTY (verify during implementation).

Remove `"charm.land/huh/v2"` import. Remove `_ "charm.land/lipgloss/v2"` blank import
if no longer needed.

### Task 1.4 — Replace huh confirm in cmd/state.go

File: `cmd/state.go`

Same pattern as Task 1.3. Find `huh.NewConfirm()` in `runStateRemoveByID()`.
Replace with `pterm.DefaultInteractiveConfirm.Show()`.
Remove `"charm.land/huh/v2"` import.

### Task 1.5 — Verify Phase 1

```bash
go build ./...
go test ./...
dotisan plan                          # spinner animates, succeeds/fails correctly
dotisan apply                         # pterm confirm prompt appears, Y/n works
dotisan state remove HomeBrewPackages/x   # pterm confirm prompt, Ctrl+C cancels
dotisan plan -o json                  # JSON output unaffected
```

---

## Phase 2: Replace lipgloss + tree + custom table

**Goal:** Remove the last charm* dependency (lipgloss). Replace the custom table renderer,
the tree renderer, and all style definitions with pterm equivalents.

**Risk:** Medium. Visual output changes (cosmetic). 6 files touched. Some lipgloss features
(arbitrary border sides) need workarounds with pterm boxes.

### Task 2.1 — Replace pkg/style/styles.go with pterm styles

File: `pkg/style/styles.go`

Map each lipgloss style to a pterm equivalent:
- `lipgloss.NewStyle().Foreground(color).Bold(true)` → `pterm.NewStyle(pterm.FgGreen, pterm.Bold)`
- `lipgloss.NewStyle().Render(text)` → `style.Sprint(text)`
- Box styles (SuccessBox, ErrorBox, etc.) → `pterm.DefaultBox.WithTitle(...).Sprint(text)`
- Header style → `pterm.DefaultHeader.Sprint(text)` or custom pterm style

Exported vars (`Success`, `Error`, `Warning`, `Info`, `Dim`, `Bold`, `Header`,
`IconSuccess`, `IconError`, etc.) must keep the same names — they are called from
`cmd/` and `pkg/engine/`.

### Task 2.2 — Replace pkg/ui/styles.go with pterm styles

File: `pkg/ui/styles.go`

Replace lipgloss color constants and style objects with pterm equivalents.
`StateAdd`, `StateRemove`, `StateUpdate`, `StateDrift`, `StateSync`, `InfoStyle`,
`WarnStyle` must remain as exported names — used in `pkg/ui/table.go` and `pkg/ui/resources.go`.

Map to `*pterm.Style` values using `pterm.NewStyle(pterm.FgColor)`.

### Task 2.3 — Replace pkg/ui/table.go with pterm.DefaultTable adapter

File: `pkg/ui/table.go`

The current custom renderer handles: fixed-width columns, flex columns, per-cell styles,
content-driven widths, ANSI-safe truncation. Replace with a thin wrapper around
`pterm.DefaultTable`.

pterm table auto-sizes columns by default. The `Column`, `Cell`, `Row`, `Table` types and
`RenderPlain(width)` method can be removed. The public surface used by `pkg/ui/resources.go`
is only `NewTable()` and `RenderPlain()` — these will be replaced in Task 2.4.

Remove `pkg/ui/table.go` entirely or reduce to a stub if anything imports it besides
`resources.go`.

### Task 2.4 — Replace pkg/ui/resources.go table building with pterm.TableData

File: `pkg/ui/resources.go`

Current: builds `[]ui.Row` with `Cell{Text, Style}` objects, calls `table.RenderPlain(width)`.

New approach:
1. Build `pterm.TableData` where first row is the header
2. Pre-color the status icon cell: `pterm.FgGreen.Sprint(icon)` for adds, `pterm.FgRed.Sprint(icon)` for removes, etc.
3. Call `pterm.DefaultTable.WithHasHeader().WithData(data).Srender()` and return the string

`RenderResourceTable(width int, rows []ResourceRow, showHeader bool) string` signature stays
the same. The `width` parameter can be ignored — pterm auto-sizes. `showHeader` maps to
`WithHasHeader()`.

### Task 2.5 — Replace pkg/diff/style.go with pterm styles

File: `pkg/diff/style.go`

Replace `Styles` struct fields (lipgloss.Style) with pterm style equivalents.
`DefaultStyles()` function must keep the same return shape — used by `pkg/diff/tree.go`
and `pkg/diff/*.go` for diff line coloring.

### Task 2.6 — Replace pkg/diff/tree.go with pterm.DefaultTree

File: `pkg/diff/tree.go`

Current: uses `lipgloss/tree` (`tree.Root()`, `tree.New()`, `tree.RoundedEnumerator`).
Replace with `pterm.DefaultTree.WithRoot(pterm.NewTreeNode(...).AddChildren(...))`.

`TreeFormatter` struct and its public methods must keep the same signatures:
- `FormatGroupPlanAsTree(info GroupPlanInfo) string`
- `FormatStateAsTree(resources []StateResource) string`

### Task 2.7 — Verify Phase 2

```bash
go build ./...
go test ./...
dotisan plan                    # pterm table, colored status icons
dotisan plan -o tree            # pterm tree renders correctly
dotisan state list              # table renders, no truncation
dotisan plan -o json            # JSON unchanged
```

---

## Phase 3: Progress Bar (Bonus)

**Goal:** Use `pterm.DefaultProgressbar` to implement the apply progress display
described in taskmaster task 9 (Enhance Apply Progress Display). This phase depends on
Phase 1 + Phase 2 being complete.

### Task 3.1 — Implement apply progress with pterm.DefaultProgressbar

File: `pkg/engine/progress.go` (or create it)

Use `pterm.DefaultProgressbar.WithTotal(n).Start()` for the apply loop.
Increment with `bar.Increment()` per item. Show success/failure counts at end.

This replaces the planned bubbletea-based progress model with a simpler pterm approach
that requires no event loop.

---

## Success Criteria

After Phase 1:
- `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/huh/v2` absent from `go.mod`
- Spinner shows animated dots (not `|/-\`)
- Confirm prompt uses pterm style

After Phase 2:
- `charm.land/lipgloss/v2` absent from `go.mod`
- No charm* imports anywhere in the codebase
- All tests pass
- Visual output equivalent or better

After Phase 3:
- `dotisan apply` shows live progress bar during package installation
