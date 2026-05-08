# Architecture Decision Record: nim

## Project Overview

**nim** is a declarative dotfiles and package manager CLI for macOS. It manages system configuration through YAML resource declarations, applying them idempotently (plan → apply → state). Think Terraform for your dotfiles and package installations.

**Author:** Piotr Boruc  
**Stack:** Go 1.26, Cobra CLI, Charmbracelet ecosystem (lipgloss v2, bubbletea, bubbles, huh)  
**Platform:** macOS only  
**State backends:** Local JSON (`~/.config/nim/state.json`) or S3-compatible

---

## Core Architecture

### 3-Layer Hierarchy

All resources follow a strict 3-level hierarchy:

```
Kind (e.g., HomeBrewPackages)
  └── Group (e.g., "core-tools")
        └── Item (e.g., "ripgrep")
```

This mirrors Kubernetes resource model patterns. Resource IDs are formatted as `Kind/Group/Item`.

### Primary Workflow

```
YAML Resources → Engine.Plan() → GroupPlan[] → Engine.ApplyWithProgress() → State Update
```

1. **Load**: Config + templates rendered (two-pass: values.yaml first, then resource files)
2. **Reconcile**: Each provider diffs desired state vs current system state
3. **Plan**: Returns additions, removals, modifications, drifted items per provider
4. **Apply**: Executes changes with live progress display; saves state on success
5. **State**: JSON document tracks what nim manages (prevents orphaned resources)

---

## Package Structure

| Package | Responsibility |
|---|---|
| `cmd/` | Cobra CLI commands (plan, apply, init, doctor, state) |
| `pkg/engine/` | Core orchestration: Plan/Apply/StateMove workflows |
| `pkg/provider/` | Provider interface, registry, and (planned) shared reconcile helpers |
| `pkg/providers/` | Concrete providers: brew, npm, go, cargo, file |
| `pkg/resource/` | Resource type definitions, YAML unmarshaling, kind-based dispatch |
| `pkg/config/` | Config loading, Go template engine with Sprig functions |
| `pkg/state/` | State persistence (local JSON or S3 backend) |
| `pkg/diff/` | Diff generation, plan formatting (plain, tree, JSON) |
| `pkg/style/` | Centralized lipgloss v2 styles, confirmation prompts, spinner helpers |
| `pkg/output/` | Output format constants (plain, tree, json) |
| `pkg/cmdutil/` | Executable checking utility |

---

## Key Interfaces

### Provider Interface (`pkg/provider/provider.go`)
```go
type Provider interface {
    Name() string
    Reconcile(desired []resource.ResourceGroup, state ResourceState) ([]GroupPlan, error)
    Apply(plan GroupPlan) error
    CheckAvailable() error
    GetState(kind, group string) (ResourceState, error)
}
```
Providers are registered in a global registry and looked up by resource Kind.

### StateBackend Interface (`pkg/state/state.go`)
```go
type StateBackend interface {
    Load() (*State, error)
    Save(state *State) error
}
```
Implementations: `LocalBackend` (default), `S3Backend` (remote sync).

### Resource Interface (`pkg/resource/resource.go`)
```go
type Resource interface {
    GetKind() string
    GetGroup() string
    GetName() string
    Validate() error
    GetItems() []string
}
```
Kind-based dynamic dispatch in `pkg/resource/unmarshal.go`.

---

## Providers

| Provider | Resource Kind | Installation Command |
|---|---|---|
| BrewProvider | `HomeBrewPackages` | `brew install` / `brew install --cask` |
| NpmProvider | `NpmPackages` | `npm install -g` |
| GoProvider | `GoPackages` | `go install` |
| CargoProvider | `CargoPackages` | `cargo install` |
 | FileProvider | `ManagedFile` (ManagedDirectory removed) | File copy/symlink/sync |

**Known issue (under refactor):** Brew, Npm, Go, Cargo providers have 90-95% identical Reconcile() implementations. Extraction to `pkg/provider/reconcile.go` is planned (Task R1 in active PRD).

---

## UI & Charmbracelet Ecosystem

| Library | Version | Usage |
|---|---|---|
| `charm.land/lipgloss/v2` | v2.0.3 | All terminal styling; styles centralized in `pkg/style/styles.go` |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | Progress model during `apply` only |
| `github.com/charmbracelet/bubbles` | v1.0.0 | `progress.Model` for apply progress bar |
| `github.com/charmbracelet/huh` | planned | Replace custom confirm prompt |

**Apply progress model:** `applyProgressModel` in `pkg/engine/engine.go` (planned move to `pkg/engine/progress.go`) implements full bubbletea Init/Update/View. Runs in goroutine while providers execute concurrently.

**Confirmation prompt:** Currently custom `ReadSingleKey()` using `golang.org/x/term` in `pkg/style/confirm.go`. Planned replacement with `huh.NewConfirm()`.

**Style system:** All CLI colors and styles are now defined in `pkg/style/styles.go`, using the [pterm](https://github.com/pterm/pterm) library. The primary palette now uses named pterm color constants: `pterm.FgGreen` (success), `pterm.FgRed` (error), `pterm.FgYellow` (warning/info), and `pterm.FgBlue` (highlight). Custom palette indices are retained only for nonstandard tints: orange (216), gray (245), and alternate row styles. All former Lipgloss v2 tree/table renderers are fully replaced with direct pterm output for consistency and maintainability.

---

## Configuration

**Config file:** `~/.config/nim/config.yaml`  
**Resource files:** `~/.config/nim/resources/` (any YAML files in this dir)  
**Template engine:** Two-pass Go templates with Sprig functions
- Pass 1: Render `values.yaml` (provides `.Values`)
- Pass 2: Render resource YAMLs with `.Values`, `.Env`, `.OS` context

**State file:** `~/.config/nim/state.json` (local backend default)

---

## Active Refactoring (PRD: prd-refactor-polish.md)

**Phase 1 — Code Quality:**
- R1: Extract shared provider reconcile helpers to `pkg/provider/reconcile.go`
- R2: Define constants for all resource kind strings and "default" namespace (40+ hardcoded occurrences)
- R3: Decompose `pkg/engine/engine.go` (979 lines) into plan.go, cleanup.go, progress.go
- R4: Fix silently swallowed errors in providers and cmd layer
- R5: Remove AI-generated debug log artifacts from brew.go, resource/brew.go, loader.go

**Phase 2 — Charm Ecosystem:**
- C1: Add huh; replace custom confirm with `huh.NewConfirm()`
- C2: Add bubbles spinner for doctor/init long-running ops
- C3: Enhance apply progress (elapsed time, task labels, 5-item rolling log, final summary box)
- C4: Centralize key bindings with `bubbles/key`

**Phase 3 — Visual Polish:**
- V1: Richer plan output (section dividers, color-coded summary box, "no changes" message)
- V2: Doctor output with grouped sections, icons, summary box
- V3: State list color-coding, welcome banner, `IsTerminal()` guard

---

## Known Technical Debt

1. **Provider duplication** — Reconcile/compareGroupItems copied 4x (brew/npm/go/cargo)
2. **engine.go monolith** — 979 lines; Plan, Apply, TUI, cleanup all in one file
3. **Hardcoded strings** — "HomeBrewPackages" (formerly "BrewPackages") etc. appear 40+ times as raw literals
4. **Two Apply implementations** — `engine.go:ApplyWithProgress()` and `apply.go:Apply()` overlap
5. **Debug artifacts** — AI-generated debug prints in brew.go (4 locations), resource/brew.go, loader.go
6. **Silent errors** — getInstalledPackages() returns empty map with no log in go/npm/cargo providers
7. **S3 no locking** — S3 backend has no state locking (concurrent apply risk)

---

## Testing

- Test files exist for: `pkg/config/`, `pkg/resource/`, `pkg/state/`, `pkg/diff/`, `pkg/providers/brew`
- `cmd/` has minimal test coverage (state_test.go is near-empty)
- No integration tests between engine and providers
- Run: `go test ./...`
