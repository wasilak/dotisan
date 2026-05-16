# Technology Stack

**Analysis Date:** 2026-05-16

## Languages

**Primary:**
- Go 1.26.2 - Entire application (`main.go`, `cmd/`, `pkg/`)

**Secondary:**
- Ruby - Homebrew formula (`Formula/nim.rb`)

## Runtime

**Environment:**
- Go runtime (CGO disabled; fully static binaries)
- No browser or server runtime — this is a CLI tool

**Package Manager:**
- Go modules (`go.mod` / `go.sum`)
- Lockfile: `go.sum` present and committed

## Frameworks

**Core:**
- `github.com/spf13/cobra` v1.10.2 - CLI command framework (all subcommands in `cmd/`)

**UI / TUI:**
- `charm.land/bubbletea/v2` v2.0.6 (indirect) - Terminal UI framework (via treeview)
- `charm.land/lipgloss/v2` v2.0.3 (indirect) - Terminal styling
- `charm.land/bubbles/v2` v2.1.0 (indirect) - UI components
- `github.com/briandowns/spinner` v1.23.2 - Spinner animations (`pkg/ui/spinner.go`)
- `github.com/aquasecurity/table` v1.11.0 - Table rendering (`pkg/ui/table.go`)
- `github.com/Digital-Shane/treeview/v2` v2.0.0 - Tree-view rendering (`pkg/diff/tree.go`)
- `github.com/fatih/color` v1.19.0 (indirect) - Terminal colors

**Diff / Syntax Highlighting:**
- `github.com/alecthomas/chroma/v2` v2.24.1 - Syntax highlighting (`pkg/diff/highlighter.go`)
- `github.com/martinohmann/go-difflib` v1.1.0 - Diff computation (`pkg/diff/diff.go`)
- `github.com/sergi/go-diff` v1.4.0 - Secondary diff library

**Templating:**
- `github.com/Masterminds/sprig/v3` v3.3.0 - Template functions for config rendering (`pkg/config/template.go`)
- `gopkg.in/yaml.v3` v3.0.1 - YAML parsing for config files

**Validation:**
- `github.com/go-playground/validator/v10` v10.30.2 - Struct validation

**Storage (S3):**
- `github.com/minio/minio-go/v7` v7.1.0 - S3-compatible state backend (`pkg/state/s3.go`)

**Build/Dev:**
- `dagger.io/dagger` v0.20.8 - CI pipeline (containerized build/test/vet in `ci/`)
- Dagger CLI v0.20.x - Orchestrates pipeline stages

**Testing:**
- Go standard `testing` package - All unit and integration tests
- No third-party assertion library (stdlib only)

## Key Dependencies

**Critical:**
- `github.com/spf13/cobra` v1.10.2 - Without this the entire CLI collapses
- `github.com/minio/minio-go/v7` v7.1.0 - Required for S3 state backend
- `github.com/Masterminds/sprig/v3` v3.3.0 - Required for config template rendering
- `gopkg.in/yaml.v3` v3.0.1 - Required for all YAML config parsing
- `github.com/go-playground/validator/v10` v10.30.2 - Required for resource validation

**Infrastructure:**
- `github.com/mattn/go-runewidth` v0.0.23 - Terminal width calculations for UI
- `github.com/rivo/uniseg` v0.4.7 (indirect) - Unicode segmentation for terminal output
- `golang.org/x/term` v0.43.0 - Terminal state detection

## Configuration

**Application Config:**
- File: `~/.nim/config.yaml` (loaded by `pkg/config/config.go`)
- Format: YAML with env var expansion
- Key fields:
  - `dotfiles_root` (default: `~/.config/nim`)
  - `state.backend` — `"local"` or `"s3"`
  - `state.path` — path for local backend
  - `state.s3.*` — endpoint, bucket, key, region, credentials
  - `ui.output` — `plain`, `tree`, or `json`
  - `log_level` — `debug`, `info`, `warn`, `error`

**Values File:**
- File: `~/.config/nim/values.yaml`
- Used as template variables when rendering resource config templates

**Build:**
- `go.mod` / `go.sum` in project root (application)
- `ci/go.mod` / `ci/go.sum` (separate module for Dagger CI pipeline)
- No `.nvmrc`, `.python-version`, or other language version files
- `Makefile` — thin wrapper over Dagger pipeline invocations

## Platform Requirements

**Development:**
- Go 1.26.2+
- Dagger CLI (for `make build` / `make test`)
- `brew` (optional, for Homebrew provider testing)

**Production / Distribution:**
- Static binaries, CGO disabled (`CGO_ENABLED=0`)
- Cross-compiled targets: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`
- Distributed via: GitHub Releases (zip archives), Homebrew tap (`Formula/nim.rb`)
- No containerized runtime — pure CLI binary

---

*Stack analysis: 2026-05-16*
