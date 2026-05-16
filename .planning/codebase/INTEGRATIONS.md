# External Integrations

**Analysis Date:** 2026-05-16

## APIs & External Services

**Homebrew API:**
- Service: Homebrew HTTP API (`https://formulae.brew.sh` indirectly via `brew info --json=v2`)
- Used for: querying installed formula/cask metadata and alias resolution
- SDK/Client: invokes `brew` CLI subprocess via `pkg/cmdutil/cmdutil.go`
- Auth: none

**Homebrew CLI:**
- Service: Local `brew` executable
- Used for: install/uninstall/tap/untap of formulae, casks, and taps
- SDK/Client: subprocess calls in `pkg/providers/brew.go`
- Auth: none (local system tool)

**Cargo / crates.io:**
- Service: `cargo install` resolves packages from `https://crates.io`
- Used for: installing Rust crates as global binaries
- SDK/Client: subprocess calls in `pkg/providers/cargo.go`
- Auth: none

**Go module proxy:**
- Service: `https://proxy.golang.org` (default Go module proxy)
- Used for: `go install <pkg>@<version>` in the Go provider
- SDK/Client: invokes `go` binary subprocess in `pkg/providers/go.go`
- Auth: none

**npm registry:**
- Service: `https://registry.npmjs.org`
- Used for: installing global npm packages
- SDK/Client: subprocess calls in `pkg/providers/npm.go`
- Auth: none

**AI Skills CLI (npx skills):**
- Service: GitHub-hosted skill packages installed via `npx skills` (npm package `skills`)
- Used for: installing AI agent skills from GitHub repositories
- SDK/Client: subprocess calls via `npx` in `pkg/providers/aiskill.go`
- Auth: none (public GitHub repos); npx resolved at runtime

## Data Storage

**Databases:**
- None — nim is stateless at the database level

**State Backends:**

*Local filesystem (default):*
- Type: JSON file
- Default path: `~/.config/nim/state.json`
- Client: Go stdlib `os`/`encoding/json`
- Config: `state.backend: local`, `state.path: <path>` in `~/.nim/config.yaml`
- Implementation: `pkg/state/local.go`

*S3-compatible object storage (optional):*
- Type: AWS S3 or any S3-compatible endpoint (MinIO, Cloudflare R2, etc.)
- Client: `github.com/minio/minio-go/v7`
- Config keys: `state.s3.endpoint`, `state.s3.bucket`, `state.s3.key`, `state.s3.region`, `state.s3.access_key_id`, `state.s3.secret_access_key`
- Credentials: stored in `~/.nim/config.yaml` (plain text — no env var injection yet)
- Implementation: `pkg/state/s3.go`

**File Storage:**
- Local filesystem only (for managed dotfiles)
- Dotfiles root defaults to `~/.config/nim`; configurable via `dotfiles_root` in `~/.nim/config.yaml`
- SHA-256 checksums used to detect file drift (`pkg/providers/file.go`)

**Caching:**
- None at application level
- Go module cache (`/go/pkg/mod`) and build cache (`/root/.cache/go-build`) are cached in Dagger CI volumes

## Authentication & Identity

**Auth Provider:**
- None — nim is a local CLI with no user accounts or API tokens
- S3 backend uses static credentials (access key + secret) stored in config file

## Monitoring & Observability

**Error Tracking:**
- None — no external error reporting service

**Logs:**
- Go standard `log/slog` package
- Output to stderr; handler format is text (default) or JSON (when `ui.output: json`)
- Level controlled via `--log-level` flag or `log_level` in config

## CI/CD & Deployment

**Hosting:**
- GitHub Releases — binary zip archives for all 4 targets
- Homebrew tap — `Formula/nim.rb` in this same repository

**CI Pipeline:**
- Service: GitHub Actions (`.github/workflows/ci.yml`)
- Trigger: push to `main`, pull requests to `main`, version tags (`v*`)
- Steps: checkout → setup-go → install Dagger CLI → run Dagger pipeline (vet + test + build)
- Artifact upload: binaries uploaded as GitHub Actions artifact on tag pushes

**Release Automation:**
- Service: GitHub Actions (`.github/workflows/ci.yml`, job `release`)
- Trigger: version tag (`v*`)
- Creates GitHub Release with `softprops/action-gh-release@v3`; attaches `dist/*.zip`

**Homebrew Formula Updates:**
- Service: GitHub Actions (`.github/workflows/homebrew.yml`)
- Trigger: GitHub release published or tag push
- Downloads release tarball, computes SHA256, verifies build via Dagger, updates `Formula/nim.rb`, commits and pushes to `main`

**Dependency Updates:**
- Service: Renovate Bot (`renovate.json`)
- Config: auto-merge enabled, separate major/minor PRs, `gomodTidy` post-update for Go modules

**Container Registry:**
- None — no Docker images produced or published

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Environment Configuration

**Required env vars (S3 backend only):**
- No env var injection implemented yet — credentials are stored directly in `~/.nim/config.yaml`
- `state.s3.access_key_id` and `state.s3.secret_access_key` fields in config

**Secrets location:**
- `~/.nim/config.yaml` — user home directory, not version-controlled
- No `.env` file, no secrets management tooling

---

*Integration audit: 2026-05-16*
