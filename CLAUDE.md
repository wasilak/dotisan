# Claude Code Instructions

## Task Master AI Instructions
**Import Task Master's development workflow commands and guidelines, treat as if import is in the main CLAUDE.md file.**
@./.taskmaster/CLAUDE.md

<!-- GSD:project-start source:PROJECT.md -->
## Project

**Nim**

Nim is a declarative dotfiles and machine configuration manager for personal use. It reads YAML resource manifests, computes a diff against persisted state, and applies changes through typed providers — bringing Terraform's plan/apply workflow to dotfiles. The current milestone adds **namespace support**, enabling different sets of resources on different machines (e.g. work vs. personal).

**Core Value:** A user can safely apply a different set of resources on each machine without duplicating manifests, by declaring a namespace on any resource and setting the active namespace via env var or CLI flag.

### Constraints

- **Compatibility**: Resources with no `metadata.namespace` must continue to work exactly as today when no namespace is set — zero breaking changes for existing dotfiles configs
- **Tech stack**: Go 1.26, Cobra CLI, Sprig templates — no new dependencies preferred
- **Testing**: stdlib `testing` only; table-driven tests following existing patterns in `pkg/providers/file_test.go` and `pkg/resource/generator_test.go`
- **Architecture**: No `context.Background()` in `pkg/`; wrap all errors with `fmt.Errorf("...: %w", err)`
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- Go 1.26.2 - Entire application (`main.go`, `cmd/`, `pkg/`)
- Ruby - Homebrew formula (`Formula/nim.rb`)
## Runtime
- Go runtime (CGO disabled; fully static binaries)
- No browser or server runtime — this is a CLI tool
- Go modules (`go.mod` / `go.sum`)
- Lockfile: `go.sum` present and committed
## Frameworks
- `github.com/spf13/cobra` v1.10.2 - CLI command framework (all subcommands in `cmd/`)
- `charm.land/bubbletea/v2` v2.0.6 (indirect) - Terminal UI framework (via treeview)
- `charm.land/lipgloss/v2` v2.0.3 (indirect) - Terminal styling
- `charm.land/bubbles/v2` v2.1.0 (indirect) - UI components
- `github.com/briandowns/spinner` v1.23.2 - Spinner animations (`pkg/ui/spinner.go`)
- `github.com/aquasecurity/table` v1.11.0 - Table rendering (`pkg/ui/table.go`)
- `github.com/Digital-Shane/treeview/v2` v2.0.0 - Tree-view rendering (`pkg/diff/tree.go`)
- `github.com/fatih/color` v1.19.0 (indirect) - Terminal colors
- `github.com/alecthomas/chroma/v2` v2.24.1 - Syntax highlighting (`pkg/diff/highlighter.go`)
- `github.com/martinohmann/go-difflib` v1.1.0 - Diff computation (`pkg/diff/diff.go`)
- `github.com/sergi/go-diff` v1.4.0 - Secondary diff library
- `github.com/Masterminds/sprig/v3` v3.3.0 - Template functions for config rendering (`pkg/config/template.go`)
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing for config files
- `github.com/go-playground/validator/v10` v10.30.2 - Struct validation
- `github.com/minio/minio-go/v7` v7.1.0 - S3-compatible state backend (`pkg/state/s3.go`)
- `dagger.io/dagger` v0.20.8 - CI pipeline (containerized build/test/vet in `ci/`)
- Dagger CLI v0.20.x - Orchestrates pipeline stages
- Go standard `testing` package - All unit and integration tests
- No third-party assertion library (stdlib only)
## Key Dependencies
- `github.com/spf13/cobra` v1.10.2 - Without this the entire CLI collapses
- `github.com/minio/minio-go/v7` v7.1.0 - Required for S3 state backend
- `github.com/Masterminds/sprig/v3` v3.3.0 - Required for config template rendering
- `gopkg.in/yaml.v3` v3.0.1 - Required for all YAML config parsing
- `github.com/go-playground/validator/v10` v10.30.2 - Required for resource validation
- `github.com/mattn/go-runewidth` v0.0.23 - Terminal width calculations for UI
- `github.com/rivo/uniseg` v0.4.7 (indirect) - Unicode segmentation for terminal output
- `golang.org/x/term` v0.43.0 - Terminal state detection
## Configuration
- File: `~/.nim/config.yaml` (loaded by `pkg/config/config.go`)
- Format: YAML with env var expansion
- Key fields:
- File: `~/.config/nim/values.yaml`
- Used as template variables when rendering resource config templates
- `go.mod` / `go.sum` in project root (application)
- `ci/go.mod` / `ci/go.sum` (separate module for Dagger CI pipeline)
- No `.nvmrc`, `.python-version`, or other language version files
- `Makefile` — thin wrapper over Dagger pipeline invocations
## Platform Requirements
- Go 1.26.2+
- Dagger CLI (for `make build` / `make test`)
- `brew` (optional, for Homebrew provider testing)
- Static binaries, CGO disabled (`CGO_ENABLED=0`)
- Cross-compiled targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Distributed via: GitHub Releases (zip archives), Homebrew tap (`Formula/nim.rb`)
- No containerized runtime — pure CLI binary
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Naming
| Symbol | Convention | Example |
|--------|-----------|---------|
| Files | `snake_case` | `apply_dag.go` |
| Packages | lowercase, no underscores | `package engine` |
| Exported types | `PascalCase` | `type ResourceState struct` |
| Unexported | `camelCase` | `func ensureProvidersRegistered()` |
| Constructors | `New<Type>(ctx, ...) (*T, error)` | `func NewEngine(ctx, cfg) (*Engine, error)` |
| Interfaces | noun or `<Verb>er` | `Provider`, `StateBackend` |
| Test fakes | `fake<Type>`, `mock<Type>`, `noop<Type>` | `fakeProvider`, `noopStateBackend` |
## Error Handling
- Always wrap with `fmt.Errorf("operation context: %w", err)` — never discard the error chain
- Use `os.IsNotExist(err)` for graceful defaults (e.g., missing state file → fresh state)
- No panics in library code (`pkg/`) — return errors to the caller
- Sentinel errors defined as package-level `var` where callers need to match with `errors.Is`
## Logging
- Package: `log/slog` (structured, key-value)
- Levels: `Debug`, `Info`, `Warn` — `Error` rare (prefer returning errors)
- Key-value pairs, not format strings: `slog.Info("applying resource", "kind", res.Kind, "name", res.Name)`
- Logging **forbidden** in `pkg/graph/` and `pkg/state/` — caller's responsibility
- `log_level` configured via `~/.nim/config.yaml`
## Comments & Documentation
- Every file in `pkg/` opens with a package doc comment: `// Package engine provides ...`
- All exported symbols have godoc comments
- Inline comments only for non-obvious logic — not for restating what the code does
## Context Usage
- `context.Context` is **always the first parameter** of any function that does I/O
- `context.Background()` is **banned in `pkg/`** — enforced by `tools/check_context_usage_test.go`
- All provider methods accept and respect context for cancellation
## Build Tags
- Integration tests: `//go:build integration` at top of file
- Integration test files: named `*_integration_test.go`
- Run with: `go test -tags integration ./...`
## Import Ordering
## Code Style
- `gofmt` / `goimports` enforced — CI runs `go vet ./...`
- No linter config (`.golangci.yml`) found — `go vet` is the enforced gate
- Short variable names (`r`, `res`, `cfg`) acceptable in short scopes; descriptive names in long functions
- Prefer early returns over deep nesting
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern
## Layers
```
```
## Entry Points
- `main.go` — wires Cobra root command; no business logic
- `cmd/root.go` — registers subcommands: `plan`, `apply`, `state`, `version`
- `cmd/plan.go` — loads config → builds engine → calls `engine.Plan()`
- `cmd/apply.go` — loads config → builds engine → calls `engine.Apply()`
## Data Flow
### Plan flow
```
```
### Apply flow
```
```
## Resource Model
```yaml
```
## Dependency Graph
## Template Rendering
## State Backends
| Backend | Implementation | Config |
|---------|---------------|--------|
| Local JSON | `pkg/state/local.go` | `state.backend: local` |
| S3-compatible | `pkg/state/s3.go` | `state.backend: s3` |
## Abstractions
```go
```
- `pkg/providers/file.go` — managed dotfiles
- `pkg/providers/brew.go` — Homebrew formulae
- `pkg/providers/cask.go` — Homebrew casks
- `pkg/providers/tap.go` — Homebrew taps
- `pkg/providers/npm.go` — npm global packages
- `pkg/providers/go.go` — `go install` packages
- `pkg/providers/cargo.go` — `cargo install` packages
- `pkg/providers/aiskill.go` — AI skill resources
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
