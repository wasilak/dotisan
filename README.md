# Dotisan 🏠⚡

> **Declarative dotfiles management for developers who treat their environment like infrastructure.**

[![Go Version](https://img.shields.io/badge/go-1.26+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Dotisan brings Terraform-inspired workflows to your dotfiles. Declare your desired state in YAML, compute diffs, and apply changes—including clean removals when you uninstall tools.

```bash
# See what would change
$ dotisan plan

# Apply with confidence
$ dotisan apply --confirm
```

## Why Dotisan?

| Feature | Chezmoi | Dotisan |
|---------|---------|---------|
| Forward sync | ✅ | ✅ |
| **Clean removals** | ❌ | ✅ |
| **Package management** | ❌ | ✅ |
| Drift detection | ❌ | ✅ |
| State tracking | ❌ | ✅ |

**The problem:** Traditional dotfile managers (chezmoi, stow, etc.) apply changes forward but never clean up. Install a tool, add it to your config, and it stays forever—even after you stop using it.

**The solution:** Dotisan tracks every managed resource with explicit state. When you remove something from your config, dotisan *removes* it from your system.

---

## Quickstart 🚀

### 1. Install Dotisan

```bash
# From source (requires Go 1.26+)
go install github.com/wasilak/dotisan@latest

# Or download a release
curl -fsSL https://github.com/wasilak/dotisan/releases/latest/download/dotisan-$(uname -s)-$(uname -m) \
  -o /usr/local/bin/dotisan && chmod +x /usr/local/bin/dotisan
```

### 2. Initialize Your Configuration

```bash
# Create the default configuration directory and files
dotisan init

# Edit your personal values
code ~/.config/dotisan/values.yaml
```

This creates:
```
~/.config/dotisan/
├── config.yaml          # Tool configuration
├── values.yaml          # Your personal variables
└── resources/           # Resource YAML files
    └── sample.yaml      # Example (remove when ready)
```

### 3. Define Your First Resource

Create `~/.config/dotisan/resources/shell.yaml`:

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: zshrc
spec:
  # Option A: Inline source (use | for multi-line)
  source: |
    # My awesome zsh config
    export EDITOR=vim
    export EMAIL={{ .Values.email }}
    
    # Load custom aliases
    [ -f ~/.aliases ] && source ~/.aliases
  
  # Option B: External file (better for IDE support)
  # sourceFile: shell/zshrc.sh
  
  destination: ~/.zshrc
  mode: "0644"
  template: true
```

### 4. Plan and Apply

```bash
# Preview changes (dry-run)
$ dotisan plan

  ManagedFile/shell/zshrc
  + destination: ~/.zshrc
  + mode: 0644

  Plan: 1 to add, 0 to change, 0 to remove

# Apply the changes
$ dotisan apply --confirm

  ✓ Created ManagedFile/shell/zshrc
  ✓ State updated

  Applied: 1 added, 0 changed, 0 removed
```

### 5. Add More Resources

#### Homebrew Packages

We now provide dedicated resource kinds for Homebrew. If you're migrating from the
legacy `BrewPackages` resource, split formulae, casks and taps into the new kinds.

Note: The loader currently supports one resource per YAML file. Multi-document
YAML files (several documents separated by `---`) will only load the first
document. Create separate files for `HomeBrewPackages`, `HomeBrewCasks`, and
`HomeBrewTaps` (one kind per file) to ensure all resources are discovered.

Example migration (YAML):

```yaml
# ~/.config/dotisan/resources/homebrew-formulae.yaml
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewPackages
metadata:
  name: core-tools
spec:
  formulae:
    - name: ripgrep
    - name: fzf

# ~/.config/dotisan/resources/homebrew-casks.yaml
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewCasks
metadata:
  name: apps
spec:
  casks:
    - name: raycast

# ~/.config/dotisan/resources/homebrew-taps.yaml
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewTaps
metadata:
  name: taps
spec:
  taps:
    - name: homebrew/cask-fonts
```

#### AI Skill Packages

Installs AI agent skill packages from GitHub repositories using the `skills` CLI (requires Node.js / `npx`).

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: AISkillPackages
metadata:
  name: my-skills
spec:
  packages:
    - source: "Ar9av/obsidian-wiki"   # GitHub repo slug or full URL
      targets:                          # Optional: limit to specific agents
        - claude
        - opencode
    - source: "some-org/another-skill" # Installs for all detected agents
```

#### NPM Global Packages

```yaml
---
apiVersion: dotisan.io/v1
kind: NpmPackages
metadata:
  name: global-cli
spec:
  packages:
    - name: typescript
    - name: @angular/cli
```

#### Entire Directory

Note: The previous `ManagedDirectory` resource kind has been removed. Use `ManagedFile` generator-based manifests or list-based `files:` entries to manage multiple files or directory-like workflows. Consult the migration guide in .taskmaster/docs/generators-prd.md for examples.

---

## Supported Resources

| Kind | Description | Provider |
|------|-------------|----------|
| `ManagedFile` | Single file with templating support | Built-in |
| `HomeBrewPackages` | Homebrew formulae (preferred) | `brew` |
| `HomeBrewCasks` | Homebrew casks (preferred) | `brew` |
| `HomeBrewTaps` | Homebrew taps (preferred) | `brew` |
| `NpmPackages` | Global npm packages | `npm` |
| `GoPackages` | Go CLI tools (`go install`) | `go` |
| `CargoPackages` | Rust CLI tools (`cargo install`) | `cargo` |
| `AISkillPackages` | AI skill packages via `npx skills` | `npx` |

---

## Commands Reference

### Setup

```bash
dotisan init              # Initialize configuration directory
dotisan doctor            # Check system prerequisites
```

### Core Workflow

```bash
dotisan plan                    # Show what would change
dotisan plan --diff             # Show inline file diffs
dotisan plan --target KIND/name # Limit to a specific resource
dotisan apply                   # Dry-run (default)
dotisan apply --confirm         # Actually apply changes
dotisan apply --diff            # Show diffs during apply
```

### State Management

```bash
dotisan state list                                         # Show all managed resources
dotisan state list --output json                           # JSON output
dotisan state import HomeBrewPackages/core-tools[ripgrep]  # Import existing resource into state
dotisan state move  HomeBrewPackages/old[pkg] HomeBrewPackages/new[pkg]  # Move item between groups
dotisan state remove HomeBrewPackages/core-tools[ripgrep]  # Remove from state (aliases: rm)
dotisan state pull                                         # Download from S3 backend
dotisan state push                                         # Upload to S3 backend
```

---

## Configuration

### `~/.dotisan/config.yaml`

```yaml
# Dotfiles location (default: ~/.config/dotisan)
dotfiles_root: ~/projects/dotfiles

# State backend (local or S3)
state:
  backend: s3
  s3:
    endpoint: s3.amazonaws.com
    bucket: my-dotisan-state
    key: state.json
    region: us-east-1
    access_key_id: ${AWS_ACCESS_KEY_ID}
    secret_access_key: ${AWS_SECRET_ACCESS_KEY}
```

### Templating

Dotisan uses Go templates with [Sprig functions](https://masterminds.github.io/sprig/):

```yaml
# In values.yaml
user: "{{ .Env.USER }}"
home: "{{ .Env.HOME }}"
hostname: "{{ .OS.Hostname }}"
arch: "{{ .OS.Arch }}"

# Use defaults
editor: '{{ default "vim" .Env.EDITOR }}'
```

Available context:
- `{{ .Values }}` - From `values.yaml`
- `{{ .Env.VAR }}` - Environment variables
- `{{ .OS.Hostname }}` - System hostname
- `{{ .OS.Arch }}` - Architecture (amd64, arm64, etc.)
- `{{ .OS.GOOS }}` - Operating system (darwin, linux, etc.)

---

## Real-World Example

A complete macOS development environment:

```yaml
# ~/.config/dotisan/resources/macos.yaml
---
apiVersion: dotisan.io/v1
kind: HomeBrewPackages
metadata:
  name: dev-tools
spec:
  formulae:
    - name: git
    - name: gh
    - name: lazygit
    - name: neovim
    - name: starship
---
apiVersion: dotisan.io/v1
kind: HomeBrewCasks
metadata:
  name: dev-casks
spec:
  casks:
    - name: raycast
    - name: warp
    - name: rectangle
---
apiVersion: dotisan.io/v1
kind: HomeBrewTaps
metadata:
  name: taps
spec:
  taps:
    - name: homebrew/cask-fonts
---
apiVersion: dotisan.io/v1
kind: ManagedFile
metadata:
  name: gitconfig
spec:
  source: |
    [user]
        name = {{ .Values.full_name }}
        email = {{ .Values.email }}
    [core]
        editor = nvim
        autocrlf = input
    [init]
        defaultBranch = main
  destination: ~/.gitconfig
  mode: "0644"
  template: true
---
apiVersion: dotisan.io/v1
kind: NpmPackages
metadata:
  name: js-tools
spec:
  packages:
    - name: typescript
    - name: ts-node
    - name: prettier
    - name: eslint
```

---

## How It Works

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Config    │────▶│   Template   │────▶│   Desired   │
│   Files     │     │    Engine    │     │    State    │
└─────────────┘     └──────────────┘     └─────────────┘
                                                │
                                                ▼
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Apply     │◀────│  Diff Engine │◀────│   Current   │
│  Changes    │     │  (plan/apply)│     │    State    │
└─────────────┘     └──────────────┘     └─────────────┘
                                                │
                                         ┌──────┴──────┐
                                         │   System    │
                                         │   State     │
                                         └─────────────┘
```

1. **Load** - Read `config.yaml`, `values.yaml`, and all resource files
2. **Template** - Apply two-pass templating (values → resources)
3. **Reconcile** - Compare desired state with current system state
4. **Plan** - Show colored diff (+add, ~change, -remove)
5. **Apply** - Execute changes and update state file

---

## Comparison with Alternatives

| Tool | Philosophy | State Tracking | Package Mgmt | Drift Detection |
|------|-----------|----------------|--------------|-----------------|
| **Dotisan** | Declarative | ✅ Yes | ✅ Yes | ✅ Yes |
| Chezmoi | Imperative | ❌ No | ❌ No | ❌ No |
| Stow | Symlinks | ❌ No | ❌ No | ❌ No |
| Ansible | Push-based | ✅ Yes | ✅ Yes | ⚠️ Complex |
| Nix | Functional | ✅ Yes | ✅ Yes | ✅ Yes |
| Homebrew Bundle | Package-only | ❌ Partial | ✅ Yes | ❌ No |

---

## Development

```bash
# Clone the repo
git clone https://github.com/wasilak/dotisan.git
cd dotisan

# Build
go build -o dotisan .

# Run tests
go test ./...

# Install locally
go install .
```

---

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Roadmap

- [ ] Windows support (PowerShell provider)
- [ ] Linux package managers (apt, dnf, pacman)
- [ ] Secrets management (1Password, Bitwarden)
- [ ] GitHub Gist backend
- [ ] GUI/TUI interface
- [ ] Plugin system for custom providers

---

## License

MIT License - see [LICENSE](LICENSE) for details.

---

## Acknowledgments

- Inspired by Terraform, Kubernetes, and chezmoi
- Built with [Cobra](https://github.com/spf13/cobra), [Sprig](https://github.com/Masterminds/sprig), [briandowns/spinner](https://github.com/briandowns/spinner), [aquasecurity/table](https://github.com/aquasecurity/table), and [minio-go](https://github.com/minio/minio-go)

---

<p align="center">
  <strong>Made with ❤️ for developers who care about their environment</strong>
</p>
