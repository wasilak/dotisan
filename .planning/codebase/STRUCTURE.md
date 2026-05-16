# Directory Structure

**Analysis Date:** 2026-05-16

## Root Layout

```
nim/
├── main.go                    Entry point — wires Cobra root command
├── go.mod / go.sum            Application module (github.com/wasilak/nim)
├── Makefile                   Thin wrapper over Dagger pipeline
├── Formula/nim.rb             Homebrew tap formula
├── .github/
│   ├── workflows/             GitHub Actions: CI, release
│   └── dependabot.yml
├── ci/                        Dagger CI pipeline (separate Go module)
│   ├── main.go                Pipeline stages: build, test, vet, lint
│   ├── go.mod / go.sum
├── cmd/                       Cobra CLI commands
├── pkg/                       Core library packages
└── tools/                     Architecture enforcement tests
```

## `cmd/` — CLI Commands

```
cmd/
├── root.go                    Root Cobra command; global flags; provider registration
├── plan.go                    `nim plan` subcommand
├── apply.go                   `nim apply` subcommand
├── state.go                   `nim state` subcommand (list, show, remove)
├── version.go                 `nim version` subcommand
└── apply_cancel_integration_test.go  Integration test (//go:build integration)
```

## `pkg/` — Library Packages

```
pkg/
├── config/
│   ├── config.go              Load + validate ~/.nim/config.yaml
│   ├── engine.go              Two-pass template rendering of manifests
│   └── template.go            Sprig template function registration
├── engine/
│   ├── engine.go              Engine struct; Plan() / Apply() entrypoints
│   ├── plan.go                Diff computation logic
│   ├── apply.go               Sequential apply (non-DAG path)
│   ├── apply_dag.go           DAG-aware apply orchestration
│   └── apply_dag_test.go      Unit tests with fakeProvider
├── graph/
│   ├── graph.go               DAG construction + topological sort
│   └── graph_test.go
├── providers/
│   ├── provider.go            Provider interface + registry
│   ├── file.go                ManagedFile provider
│   ├── brew.go                HomeBrewPackages provider
│   ├── cask.go                Casks provider
│   ├── tap.go                 Taps provider
│   ├── npm.go                 NpmPackages provider
│   ├── go.go                  GoPackages provider
│   ├── cargo.go               CargoPackages provider
│   ├── aiskill.go             AISkill provider
│   ├── file_test.go
│   └── provider_test.go
├── resource/
│   ├── resource.go            Resource struct; YAML unmarshalling
│   ├── generator.go           Manifest discovery + loading from dotfiles_root
│   ├── generator_test.go      Table-driven tests
│   └── types.go               Kind constants
├── state/
│   ├── state.go               StateBackend interface + ResourceState types
│   ├── local.go               Local JSON backend (~/.nim/state.json)
│   └── s3.go                  S3-compatible backend (minio-go)
├── diff/
│   ├── diff.go                Diff computation between desired and actual state
│   ├── highlighter.go         Chroma syntax highlighting
│   ├── tree.go                Tree-view diff rendering (treeview + Lipgloss)
│   └── diff_test.go
├── ui/
│   ├── spinner.go             Spinner animations (briandowns/spinner)
│   └── table.go               Table rendering (aquasecurity/table)
└── style/
    └── style.go               Shared Lipgloss style definitions
```

## `tools/` — Enforcement

```
tools/
└── check_context_usage_test.go   Architecture lint: bans context.Background() in pkg/
```

## Key File Locations

| Purpose | Path |
|---------|------|
| App config | `~/.nim/config.yaml` |
| Values file | `~/.config/nim/values.yaml` |
| Local state | `~/.nim/state.json` (default) |
| Dotfiles root | `~/.config/nim/` (default) |
| CI pipeline | `ci/main.go` |
| Homebrew formula | `Formula/nim.rb` |

## Naming Conventions

- **Files**: `snake_case.go`
- **Test files**: `<name>_test.go` (unit), `<name>_unit_test.go` (explicit unit), `<name>_integration_test.go` (integration, build-tagged)
- **Packages**: short, lowercase, no underscores (`config`, `engine`, `providers`)
- **Types**: `PascalCase`
- **Unexported symbols**: `camelCase`
- **Constructors**: `New<Type>(ctx, ...) (*Type, error)`

---

*Structure analysis: 2026-05-16*
