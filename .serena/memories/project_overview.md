# dotisan — Project Overview

## Purpose
`dotisan` is a declarative dotfiles management CLI tool written in Go. It treats a local developer environment like Terraform treats cloud infrastructure: declare desired state in version-controlled config files, compute a diff against current state, and apply changes — including **removals**.

Unlike `chezmoi` which applies changes forward but never cleans up, dotisan tracks managed resources explicitly and handles removals as first-class operations.

## Tech Stack
- **Language**: Go
- **CLI Framework**: `github.com/spf13/cobra`
- **YAML Parsing**: `gopkg.in/yaml.v3`
- **Templating**: Go `text/template` + `github.com/Masterminds/sprig/v3` (Helm-compatible)
- **Diff Engine**: `github.com/martinohmann/go-difflib` (line-level) + `github.com/sergi/go-diff` (character-level)
- **S3 Backend**: `github.com/minio/minio-go/v7`
- **Terminal Styling**: `github.com/charmbracelet/lipgloss`
- **Validation**: `github.com/go-playground/validator/v10`

## Architecture

### Core Components
```
CLI (cobra) → Engine (plan/apply/diff) → Provider Registry
                                    ↓
                        Template Renderer (sprig)
                        Diff Engine (go-difflib + go-diff)
                        State Backend (local/S3)
```

### Providers
- `BrewProvider` — Homebrew formulae, casks, taps
- `NpmProvider` — Global npm packages
- `GoProvider` — Go modules via `go install`
- `CargoProvider` — Rust crates via `cargo install`
- `FileProvider` — Managed files and directories with templating

### State Management
Two backends:
- **LocalBackend** — JSON file at `~/.dotisan/state.json`
- **S3Backend** — S3-compatible storage via minio-go

### Config Files
- `~/.dotisan/config.yaml` — Tool configuration (backend, paths)
- `~/.dotfiles/values.yaml` — Templated values (user vars, paths)
- `~/.dotfiles/*.yaml` — Resource declarations (Kubernetes-style)

## Key Design Principles
1. **Declarative over imperative** — describe what should exist
2. **Stateful** — track managed resources; removals are first-class
3. **Dry-run by default** — `apply` shows plan, requires `--confirm`
4. **Provider-based** — isolated providers per resource type
5. **Kubernetes-style config** — typed YAML with `apiVersion`, `kind`, `metadata`, `spec`
6. **Helm-style templating** — `values.yaml` + Go templates + sprig

## Project Status
- Currently: **Pre-implementation** — 13 tasks defined, none started
- First task: Project Setup and CLI Framework Integration (Task #1)

## Resource Kinds Supported
- `BrewPackages` — Homebrew formulae, casks, taps
- `NpmPackages` — Global npm packages
- `GoPackages` — Go modules/tools
- `CargoPackages` — Rust crates
- `ManagedFile` — Single file with optional templating
- `ManagedDirectory` — Directory sync with recursive/clean options

## CLI Commands
- `dotisan plan` — Show what would change
- `dotisan apply` — Apply changes (dry-run unless `--confirm`)
- `dotisan doctor` — Check system prerequisites
- `dotisan state import/remove/list/pull/push` — State management
- `dotisan eject` — Stop managing a resource

## Development Environment
- **OS**: Darwin (macOS primary, Linux secondary)
- **Go Version**: Latest stable (to be determined at init)
- **No Windows support** (explicit non-goal)
