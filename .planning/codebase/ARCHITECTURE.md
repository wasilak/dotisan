# Architecture

**Analysis Date:** 2026-05-16

## Pattern

**Declarative CLI with Plan/Apply Model** (Terraform-inspired)

nim reads YAML resource manifests, computes a diff against persisted state, presents a plan, and applies changes through typed providers. The user never mutates state directly — all changes go through providers.

## Layers

```
CLI Layer         cmd/           Cobra commands; flag parsing; user I/O
Engine Layer      pkg/engine/    Orchestrator: plan, apply, state sync
Provider Layer    pkg/providers/ Resource-type implementations (6 types)
Graph Layer       pkg/graph/     DAG for topological dependency ordering
State Layer       pkg/state/     State persistence (local JSON or S3)
Config Layer      pkg/config/    YAML loading, template rendering (Sprig)
UI Layer          pkg/ui/        Spinner, table, diff display (Lipgloss)
```

## Entry Points

- `main.go` — wires Cobra root command; no business logic
- `cmd/root.go` — registers subcommands: `plan`, `apply`, `state`, `version`
- `cmd/plan.go` — loads config → builds engine → calls `engine.Plan()`
- `cmd/apply.go` — loads config → builds engine → calls `engine.Apply()`

## Data Flow

### Plan flow
```
cmd/plan.go
  → pkg/config/config.go       Load ~/.nim/config.yaml + values.yaml
  → pkg/config/engine.go       Two-pass Go template render of manifests
  → pkg/engine/plan.go         Load desired state from manifests
  → pkg/state/*.go             Load current state (local or S3)
  → pkg/engine/diff.go         Compute create/update/delete diff
  → pkg/ui/                    Render diff tree
```

### Apply flow
```
cmd/apply.go
  → pkg/engine/apply.go        Build DAG from resource dependsOn
  → pkg/graph/graph.go         Topological sort
  → pkg/engine/apply_dag.go    Walk DAG; invoke providers in order
  → pkg/providers/*.go         Execute resource-specific operations
  → pkg/state/*.go             Persist updated state
```

## Resource Model

Resources are Kubernetes-style YAML manifests:

```yaml
apiVersion: nim.run/v1alpha1
kind: ManagedFile
metadata:
  name: zshrc
  dependsOn:
    - HomeBrewPackages/core-tools
spec:
  source: templates/zshrc.tmpl
  destination: ~/.zshrc
```

Supported kinds: `ManagedFile`, `HomeBrewPackages`, `Casks`, `Taps`, `NpmPackages`, `GoPackages`, `CargoPackages`, `AISkill`

## Dependency Graph

`pkg/graph/graph.go` builds a DAG from `metadata.dependsOn` edges and emits a topologically sorted execution order. Cycles are detected and rejected at plan time.

## Template Rendering

`pkg/config/engine.go` performs two-pass rendering:
1. Render `values.yaml` with environment variables
2. Render each resource manifest with rendered values as template context

Sprig template functions available (`pkg/config/template.go`).

## State Backends

| Backend | Implementation | Config |
|---------|---------------|--------|
| Local JSON | `pkg/state/local.go` | `state.backend: local` |
| S3-compatible | `pkg/state/s3.go` | `state.backend: s3` |

State shape: `map[string]ResourceState` keyed by `kind/name`.

## Abstractions

**Provider interface** (`pkg/providers/provider.go`):
```go
type Provider interface {
    Plan(ctx context.Context, resource Resource) ([]Change, error)
    Apply(ctx context.Context, resource Resource) error
    Delete(ctx context.Context, resource Resource) error
}
```

**6 built-in providers:**
- `pkg/providers/file.go` — managed dotfiles
- `pkg/providers/brew.go` — Homebrew formulae
- `pkg/providers/cask.go` — Homebrew casks
- `pkg/providers/tap.go` — Homebrew taps
- `pkg/providers/npm.go` — npm global packages
- `pkg/providers/go.go` — `go install` packages
- `pkg/providers/cargo.go` — `cargo install` packages
- `pkg/providers/aiskill.go` — AI skill resources

---

*Architecture analysis: 2026-05-16*
