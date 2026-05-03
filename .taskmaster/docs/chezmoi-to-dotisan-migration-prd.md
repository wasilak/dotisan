# PRD: Chezmoi to Dotisan Migration

## 1. Executive Summary

This document outlines the complete migration of dotfiles management from **chezmoi** to **dotisan**, leveraging dotisan's Terraform-like state management and native Go templating.

### Key Benefits of Migration

| Feature | Chezmoi | Dotisan | Impact |
|---------|---------|---------|--------|
| State tracking | ❌ No | ✅ Yes | Track all managed resources |
| Clean removals | ❌ No | ✅ Yes | Remove packages/configs when deleted |
| Drift detection | ❌ No | ✅ Yes | Detect manual changes |
| Package management | ⚠️ Scripts | ✅ Native | Built-in brew/npm/go providers |
| Templating | ✅ Go templates | ✅ Go templates + Sprig | Full feature parity |
| Multi-format generation | ⚠️ Complex | ✅ Native templates | Single source, multiple outputs |

### Migration Scope

**Total Files to Migrate:** 62 tracked files
- 14 template files (.tmpl)
- 7 YAML data files
- 35+ static config files
- 4 run scripts (become unnecessary with state)

---

## 2. Current State Analysis

### 2.1 Chezmoi Repository Structure

```
/Users/piotrek/git/dotfiles/
├── .chezmoi.yaml.tmpl                    # Chezmoi config
├── .chezmoidata/                         # Data files
│   ├── agents.yaml                       # AI agent definitions
│   ├── commands.yaml                     # 7 command definitions (743 lines)
│   ├── completions.yaml                  # Shell completions
│   ├── mcp_permissions.yaml              # MCP permission matrix
│   ├── mcp_servers.yaml                  # 12 MCP server configs
│   ├── packages.yaml                     # Package lists
│   └── skills.yaml                       # Skill definitions
├── dot_agents/                           # AI agent skills (~/.agents/)
│   └── skills/*/
│       └── SKILL.md                      # 14 skill files
├── dot_claude/                           # Claude Code config (~/.claude/)
│   ├── settings.json.tmpl                # MCP permissions
│   └── CLAUDE.md                         # Static
├── dot_ssh/
│   └── config.tmpl                       # SSH config
├── dot_zshrc.tmpl                        # Zsh config
├── dot_gitconfig.tmpl                    # Git config
├── dot_taskmaster_config.json.tmpl       # TaskMaster config
├── home/                                 # Home directory files
├── Library/                              # macOS app configs
│   └── Application Support/
│       ├── Code/User/
│       │   ├── mcp.json.tmpl             # VS Code MCP
│       │   └── settings.json.tmpl        # VS Code settings
│       └── com.mitchellh.ghostty/
│           └── config                    # Ghostty config
├── private_dot_config/                   # ~/.config/ files
│   ├── fish/
│   │   └── config.fish                   # Fish shell
│   ├── opencode/
│   │   ├── opencode.json.tmpl            # OpenCode MCP
│   │   └── AGENTS.md                     # Static
│   ├── starship/
│   │   └── starship.toml                 # Prompt config
│   ├── wezterm/
│   │   └── wezterm.lua                   # Terminal config
│   └── zellij/
│       └── config.kdl.tmpl               # Multiplexer config
└── run_*.sh.tmpl                         # Scripts (4 files)
```

### 2.2 Chezmoi Features Used

| Feature | Usage Count | Migration Strategy |
|---------|-------------|-------------------|
| Template files (.tmpl) | 14 | Convert to `ManagedFile` with `template: true` |
| Data files (.chezmoidata/) | 7 | Consolidate into `values.yaml` |
| Template variables | ~50 | Map to `.Values.*` |
| Template functions | 15+ | Sprig equivalents available |
| Run scripts | 4 | **Eliminated** - state handles idempotency |
| Conditional logic | Extensive | Go template conditionals |
| Loops (range) | Extensive | Go template range |

### 2.3 Template Variables Mapping

**Chezmoi Variables → Dotisan Variables**

| Chezmoi | Dotisan | Usage |
|---------|---------|-------|
| `{{ .chezmoi.homeDir }}` | `{{ .Env.HOME }}` | Home directory |
| `{{ .chezmoi.sourceDir }}` | `{{ .Values.dotfiles_root }}` | Source directory |
| `{{ .chezmoi.os }}` | `{{ .OS.GOOS }}` | Operating system |
| `{{ .chezmoi.arch }}` | `{{ .OS.GOARCH }}` | Architecture |
| `{{ env "VAR" }}` | `{{ .Env.VAR }}` | Environment variables |
| `.chezmoidata.X` | `.Values.X` | Data files |

---

## 3. Target State Design

### 3.1 Dotisan Repository Structure

```
~/git/dotisan-dotfiles/                   # New dotfiles repo
├── .gitignore
├── README.md                             # Migration guide
├── values.yaml                           # All configuration data
│   # (consolidated from .chezmoidata/)
├── config.yaml                           # Dotisan config (optional)
└── resources/                            # Resource definitions
    ├── 00-packages/                      # Package management
    │   ├── homebrew.yaml                 # Brew formulas/casks
    │   ├── go-tools.yaml                 # Go packages
    │   └── npm-tools.yaml                # NPM packages
    ├── 10-shell/                         # Shell configs
    │   ├── zsh.yaml                      # Zsh + zshrc
    │   └── fish.yaml                     # Fish config
    ├── 20-terminals/                     # Terminal emulators
    │   ├── wezterm.yaml                  # WezTerm config
    │   └── ghostty.yaml                  # Ghostty config
    ├── 30-tools/                         # CLI tools
    │   ├── git.yaml                      # Git config
    │   ├── ssh.yaml                      # SSH config
    │   ├── zellij.yaml                   # Zellij config
    │   └── starship.yaml                 # Starship config
    ├── 40-ai-tools/                      # AI tool configs
    │   ├── opencode.yaml                 # OpenCode + MCP
    │   ├── claude-code.yaml              # Claude Code + MCP
    │   ├── vscode-mcp.yaml               # VS Code MCP
    │   └── taskmaster.yaml               # TaskMaster config
    ├── 50-agents/                        # AI agent files
    │   └── agents.yaml                   # ManagedDirectory (removed) - convert to ManagedFile generators or files lists
    └── 99-generated/                     # Generated command files
        ├── opencode-commands.yaml        # OpenCode commands
        └── claude-commands.yaml          # Claude Code commands
```

### 3.2 Data Consolidation (values.yaml)

**Consolidated structure from all .chezmoidata/ files:**

```yaml
# =============================================================================
# USER CONFIGURATION
# =============================================================================
user:
  name: "Piotrek"
  email: "piotrek@example.com"
  github_username: "wasilak"

# =============================================================================
# PATHS
# =============================================================================
paths:
  home: "{{ .Env.HOME }}"
  dotfiles: "{{ .Env.HOME }}/git/dotisan-dotfiles"
  projects: "{{ .Env.HOME }}/Projects"
  obsidian_vault: "{{ .Env.HOME }}/Documents/notes"

# =============================================================================
# PACKAGES
# =============================================================================
packages:
  # Homebrew formulas
  brew:
    formulas:
      - name: zsh
      - name: oh-my-posh
      - name: fzf
      - name: starship
      - name: coreutils
      - name: zsh-autosuggestions
      - name: zsh-syntax-highlighting
      - name: zsh-history-substring-search
      - name: thefuck
      - name: kubectl
      - name: dyff
      - name: eza
      - name: mitmproxy
      - name: zoxide
      - name: git-delta
      - name: bat
      - name: scc
      - name: sd
      - name: xh
      - name: pyenv
      - name: stern
      - name: procs
      - name: glances
      - name: keepalive
      - name: SurgeDM/tap/surge
      - name: fabric-ai
      - name: fclones
      - name: chojs23/tap/ec
      - name: dlvhdr/formulae/diffnav
    
    casks:
      - name: lazyworktree
      - name: wezterm
      - name: ghostty
      - name: font-droid-sans-mono-nerd-font
      - name: font-fira-code-nerd-font
      - name: font-fira-mono-nerd-font
      - name: font-go-mono-nerd-font
      - name: font-hasklug-nerd-font
      - name: font-inconsolata-nerd-font
      - name: font-sauce-code-pro-nerd-font
    
    taps:
      - name: stigoleg/homebrew-tap
      - name: chmouel/lazyworktree
        url: https://github.com/chmouel/lazyworktree
      - name: chojs23/tap
      - name: dlvhdr/formulae
  
  # Go packages
  go:
    - module: github.com/Gelio/go-global-update
    - module: github.com/zricethezav/gitleaks
    - module: golang.org/x/tools/cmd/deadcode
    - module: github.com/go-delve/delve/cmd/dlv
    - module: golang.org/x/tools/gopls
    - module: github.com/boyter/scc/v2/cmd/scc
  
  # NPM packages
  npm:
    - name: carbon-now-cli
    - name: freebuff
    - name: task-master-ai

# =============================================================================
# MCP SERVERS
# =============================================================================
mcp_servers:
  sequential-thinking:
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-sequential-thinking"
    type: local
  
  spec-workflow:
    command: npx
    args:
      - -y
      - "@pimzino/spec-workflow-mcp@latest"
    type: local
  
  git:
    command: uvx
    args:
      - mcp-server-git
    type: local
  
  mcp-mermaid:
    command: npx
    args:
      - -y
      - mcp-mermaid
    type: local
  
  context7:
    type: remote
    url: https://mcp.context7.com/mcp
    headers:
      CONTEXT7_API_KEY: "{env:CONTEXT7_API_TOKEN}"
    enabled: true
  
  github:
    type: remote
    url: https://api.githubcopilot.com/mcp/
    headers:
      Authorization: "Bearer {env:GITHUB_PERSONAL_ACCESS_TOKEN}"
      X-MCP-Toolsets: actions
    enabled: true
  
  fetch:
    command: uvx
    args:
      - mcp-server-fetch
    type: local
  
  serena:
    command: /opt/homebrew/bin/uvx
    args:
      - --from
      - git+https://github.com/oraios/serena
      - serena
      - start-mcp-server
      - --context
      - ide-assistant
      - --project-from-cwd
      - --mode=planning
      - --mode=editing
      - --mode=interactive
      - --open-web-dashboard=false
    type: local
  
  obsidian:
    command: npx
    args:
      - -y
      - "@bitbonsai/mcpvault@latest"
      - "{{ .Values.paths.obsidian_vault }}"
    type: local
  
  codebase-memory-mcp:
    command: "{{ .Values.paths.home }}/.local/bin/codebase-memory-mcp"
    args: []
    env:
      CBM_CACHE_DIR: "{{ .Values.paths.home }}/.cache/codebase-memory-mcp"
    type: local
  
  karakeep:
    command: npx
    args:
      - -y
      - "@karakeep/mcp"
    env:
      KARAKEEP_API_ADDR: http://karakeep.default.svc.cluster.local
      KARAKEEP_API_KEY: "{env:KARAKEEP_API_KEY}"
    type: local
  
  taskmaster:
    command: npx
    args:
      - -y
      - task-master-ai@latest
    env:
      TASKMASTER_PROVIDER: "google"
      GOOGLE_API_KEY: "{env:GOOGLE_API_KEY}"
      TASK_MASTER_TOOLS: standard
    enabled: true
    type: local

# =============================================================================
# MCP PERMISSIONS
# =============================================================================
mcp_permissions:
  trusted_servers:
    - sequential-thinking
    - git
    - fetch
    - mcp-mermaid
    - spec-workflow
    - obsidian
    - karakeep
    - taskmaster
  
  readonly_servers:
    context7:
      - query-docs
      - resolve-library-id
    github:
      - list-repositories
      - get-issue
  
  readonly_glob_servers:
    serena:
      - "get_*"
      - "list_*"

# =============================================================================
# COMMANDS
# =============================================================================
commands:
  kickoff:
    name: pb:kickoff
    shorthand: /pb:kickoff
    skill: pb-kickoff
    agent: plan
    description_short: Lightweight session start
    description_long: Verify environment readiness and load guides
    triggers:
      - kickoff
      - session start
    content: |
      ## ⚠️ READ-ONLY STATUS CHECK
      ... (full content)
  
  init-project:
    name: pb:init-project
    shorthand: /pb:init-project
    skill: pb-init-project
    # ... (additional commands)

# =============================================================================
# AGENTS
# =============================================================================
agents:
  plan:
    name: "Planning Agent"
    description: "Analyzes and plans tasks"
    skills:
      - pb-kickoff
      - pb-init-project
  
  code:
    name: "Code Agent"
    description: "Implements code changes"
    skills:
      - pb-active-dev
      - pb-tracing-symbols

# =============================================================================
# SKILLS
# =============================================================================
skills:
  pb-kickoff:
    name: "Kickoff Skill"
    description: "Session initialization"
    file: pb-kickoff.md
  
  pb-active-dev:
    name: "Active Development"
    description: "Implementation mode"
    file: pb-active-dev.md
  # ... (14 skills total)

# =============================================================================
# COMPLETIONS
# =============================================================================
completions:
  - eza
  - lla
  - kubectl
  - stern
  - task
```

---

## 4. Detailed Migration Plan

### 4.1 Phase 1: Foundation (Week 1)

**Goal:** Set up dotisan infrastructure and migrate core packages

#### Task 1.1: Initialize Dotisan Dotfiles Repository

```bash
# Create new repository
mkdir -p ~/git/dotisan-dotfiles
cd ~/git/dotisan-dotfiles
git init

# Create structure
mkdir -p resources/{00-packages,10-shell,20-terminals,30-tools,40-ai-tools,50-agents,99-generated}

# Initialize dotisan
dotisan init
```

#### Task 1.2: Create values.yaml

**Source:** Consolidate all `.chezmoidata/*.yaml` into single `values.yaml`

**Validation:**
```bash
# Test template rendering
dotisan plan
```

#### Task 1.3: Migrate Package Management

**Create:** `resources/00-packages/homebrew.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
    kind: HomeBrewPackages
metadata:
  name: core-tools
  namespace: default
spec:
  taps:
    {{- range .Values.packages.brew.taps }}
    - name: {{ .name }}
      {{- if .url }}
      url: {{ .url }}
      {{- end }}
    {{- end }}
  
  formulae:
    {{- range .Values.packages.brew.formulas }}
    - name: {{ .name }}
    {{- end }}
  
  casks:
    {{- range .Values.packages.brew.casks }}
    - name: {{ .name }}
    {{- end }}
```

**Create:** `resources/00-packages/go-tools.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: GoPackages
metadata:
  name: dev-tools
  namespace: default
spec:
  packages:
    {{- range .Values.packages.go }}
    - module: {{ .module }}
      version: latest
    {{- end }}
```

**Create:** `resources/00-packages/npm-tools.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: NpmPackages
metadata:
  name: global-cli
  namespace: default
spec:
  packages:
    {{- range .Values.packages.npm }}
    - name: {{ .name }}
    {{- end }}
```

#### Task 1.4: Test Package Migration

```bash
# Dry run
dotisan plan

# Apply if looks correct
dotisan apply --confirm

# Verify state
dotisan state list
```

### 4.2 Phase 2: Shell & Core Configs (Week 1-2)

**Goal:** Migrate shell configs and essential tool configurations

#### Task 2.1: Migrate Zsh Configuration

**Source:** `dot_zshrc.tmpl`

**Create:** `resources/10-shell/zsh.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: zshrc
  namespace: default
spec:
  sourceFile: 10-shell/zshrc.sh
  destination: ~/.zshrc
  mode: "0644"
  template: true
```

**Create:** `resources/10-shell/zshrc.sh`

```bash
# Source template content from dot_zshrc.tmpl
# Replace chezmoi variables with dotisan equivalents:
# {{ .chezmoi.homeDir }} -> {{ .Env.HOME }}
# {{ .chezmoidata.completions }} -> {{ .Values.completions }}
```

#### Task 2.2: Migrate Fish Configuration

**Source:** `private_dot_config/fish/config.fish`

**Create:** `resources/10-shell/fish.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: fish-config
  namespace: default
spec:
  sourceFile: 10-shell/config.fish
  destination: ~/.config/fish/config.fish
  mode: "0644"
```

#### Task 2.3: Migrate Git Configuration

**Source:** `dot_gitconfig.tmpl`

**Create:** `resources/30-tools/git.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: gitconfig
  namespace: default
spec:
  source: |
    [user]
        name = {{ .Values.user.name }}
        email = {{ .Values.user.email }}
    [core]
        editor = nvim
        autocrlf = input
    [init]
        defaultBranch = main
    # ... rest of config
  destination: ~/.gitconfig
  mode: "0644"
  template: true
```

### 4.3 Phase 3: Terminal Emulators (Week 2)

**Goal:** Migrate WezTerm and Ghostty configurations

#### Task 3.1: WezTerm Configuration

**Source:** `private_dot_config/wezterm/wezterm.lua`

**Create:** `resources/20-terminals/wezterm.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: wezterm-config
  namespace: default
spec:
  sourceFile: 20-terminals/wezterm.lua
  destination: ~/.config/wezterm/wezterm.lua
  mode: "0644"
```

#### Task 3.2: Ghostty Configuration

**Source:** `Library/Application Support/com.mitchellh.ghostty/config`

**Create:** `resources/20-terminals/ghostty.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: ghostty-config
  namespace: default
spec:
  sourceFile: 20-terminals/ghostty.conf
  destination: "~/Library/Application Support/com.mitchellh.ghostty/config"
  mode: "0644"
```

### 4.4 Phase 4: AI Tool Configurations (Week 2-3)

**Goal:** Migrate MCP server configs with multi-format generation

#### Task 4.1: OpenCode Configuration

**Source:** `private_dot_config/opencode/opencode.json.tmpl`

**Create:** `resources/40-ai-tools/opencode.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: opencode-config
  namespace: default
spec:
  source: |
    {{- $homeDir := .Env.HOME -}}
    {{- $servers := .Values.mcp_servers }}
    {{- $permissions := .Values.mcp_permissions }}
    {
      "$schema": "https://opencode.ai/config.json",
      "mcp": {
        {{- $serverNames := keys $servers | sortAlpha }}
        {{- range $index, $name := $serverNames }}
        {{- $server := index $servers $name }}
        {{- $skipIn := list }}
        {{- if hasKey $server "skip_in" }}{{ $skipIn = $server.skip_in }}{{ end }}
        {{- if not (has "opencode" $skipIn) }}
        {{- if gt $index 0 }},{{ end }}
        {{- if and (hasKey $server "type") (eq $server.type "remote") }}
        "{{ $name }}": {
          "type": "remote",
          "url": "{{ $server.url }}",
          {{- if hasKey $server "headers" }}
          "headers": {
            {{- $headerKeys := keys $server.headers | sortAlpha }}
            {{- range $headerIndex, $headerKey := $headerKeys }}
            {{- if gt $headerIndex 0 }},{{ end }}
            "{{ $headerKey }}": "{{ index $server.headers $headerKey }}"
            {{- end }}
          },
          {{- end }}
          "enabled": {{ if hasKey $server "enabled" }}{{ $server.enabled }}{{ else if or (has $name $permissions.trusted_servers) (hasKey $permissions.readonly_servers $name) }}true{{ else }}false{{ end }}
        }
        {{- else }}
        "{{ $name }}": {
          "type": "local",
          "command": [
            "{{ $server.command }}"
            {{- range $server.args }}
            {{- if eq . "OBSIDIAN_VAULT_PATH" }}
            , "{{ $homeDir }}/Documents/notes"
            {{- else if and (eq $name "serena") (eq . "ide-assistant") }}
            , "ide-assistant"
            {{- else }}
            , "{{ . }}"
            {{- end }}
            {{- end }}
          ],
          "enabled": {{ if or (has $name $permissions.trusted_servers) (hasKey $permissions.readonly_servers $name) }}true{{ else }}false{{ end }}
          {{- if hasKey $server "env" }},
          "environment": {
            {{- $envKeys := keys $server.env | sortAlpha }}
            {{- range $envIndex, $envKey := $envKeys }}
            {{- if gt $envIndex 0 }},{{ end }}
            "{{ $envKey }}": "{{ index $server.env $envKey }}"
            {{- end }}
          }
          {{- end }}
        }
        {{- end }}
        {{- end }}
        {{- end }}
      },
      "permission": {
        {{- $firstPermission := true }}
        {{- range $server := $permissions.trusted_servers }}
        {{- if not $firstPermission }},{{ end }}
        "{{ $server }}_*": "allow"
        {{- $firstPermission = false }}
        {{- end }}
        {{- $readonlyServers := keys $permissions.readonly_servers | sortAlpha }}
        {{- range $server := $readonlyServers }}
        {{- if not $firstPermission }},{{ end }}
        "{{ $server }}_*": "deny"
        {{- $firstPermission = false }}
        {{- $tools := index $permissions.readonly_servers $server }}
        {{- range $tool := $tools }},
        "{{ $server }}_{{ $tool }}": "allow"
        {{- end }}
        {{- end }}
      }
    }
  destination: ~/.config/opencode/opencode.json
  mode: "0644"
  template: true
```

#### Task 4.2: Claude Code Configuration

**Source:** `dot_claude/settings.json.tmpl`

**Create:** `resources/40-ai-tools/claude-code.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: claude-settings
  namespace: default
spec:
  source: |
    {{- $permissions := .Values.mcp_permissions }}
    {
      "includeCoAuthoredBy": false,
      "attribution": {
        "commit": "never",
        "pr": "never"
      },
      "permissions": {
        "allow": [
          {{- range $server := $permissions.trusted_servers }}
          "mcp__{{ lower $server }}__*",
          {{- end }}
          {{- $readonlyServers := keys $permissions.readonly_servers | sortAlpha }}
          {{- range $server := $readonlyServers }}
          {{- $tools := index $permissions.readonly_servers $server }}
          {{- range $tools }}
          "mcp__{{ lower $server }}__{{ . }}",
          {{- end }}
          {{- end }}
          {{- $readonlyGlobServers := keys $permissions.readonly_glob_servers | sortAlpha }}
          {{- range $server := $readonlyGlobServers }}
          {{- $patterns := index $permissions.readonly_glob_servers $server }}
          {{- range $pattern := $patterns }}
          "mcp__{{ lower $server }}__{{ . }}",
          {{- end }}
          {{- end }}
          "Bash(terraform plan)",
          "Bash(terraform providers lock *)",
          "Bash(gh api:*)",
          "WebFetch(domain:github.com)"
        ],
        "deny": [
          "Bash(grep *)",
          "Bash(find *)"
        ]
      },
      "enabledPlugins": {
        "gopls-lsp@claude-plugins-official": true
      }
    }
  destination: ~/.claude/settings.json
  mode: "0644"
  template: true
```

#### Task 4.3: VS Code MCP Configuration

**Source:** `Library/Application Support/Code/User/mcp.json.tmpl`

**Create:** `resources/40-ai-tools/vscode-mcp.yaml`

Similar template structure adapted for VS Code's MCP format.

### 4.5 Phase 5: AI Agent Files (Week 3)

**Goal:** Migrate AI agent skills and configurations

#### Task 5.1: Agent Skills Directory

**Source:** `dot_agents/skills/*`

**Create:** `resources/50-agents/agents.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedDirectory # REMOVED - convert to ManagedFile generator or files list
metadata:
  name: agent-skills
  namespace: default
spec:
  sourceDir: 50-agents/skills
  destination: ~/.agents/skills
  recursive: true
  clean: true
```

**Action:** Copy all skill files from `dotfiles/dot_agents/skills/` to `resources/50-agents/skills/`

### 4.6 Phase 6: Generated Commands (Week 3)

**Goal:** Migrate command generation from Python script to templates

#### Task 6.1: OpenCode Commands

**Create:** `resources/99-generated/opencode-commands.yaml`

```yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: opencode-commands
  namespace: default
spec:
  source: |
    {{- range $key, $cmd := .Values.commands }}
    ---
    # {{ $cmd.name }} - {{ $cmd.description_short }}
    # Triggers: {{ join ", " $cmd.triggers }}
    {{ $cmd.content | indent 4 }}
    {{- end }}
  destination: ~/.config/opencode/commands.md
  mode: "0644"
  template: true
```

*Note: This is a simplified example. The actual implementation may need separate files per command depending on OpenCode's requirements.*

#### Task 6.2: Claude Code Commands

**Create:** `resources/99-generated/claude-commands.yaml`

Similar structure for Claude Code command format.

---

## 5. Data Mapping Reference

### 5.1 Complete Variable Mapping

| Source File | Variable Path | Dotisan Path |
|-------------|---------------|--------------|
| `.chezmoidata/packages.yaml` | `.brew_packages` | `.Values.packages.brew.formulas` |
| `.chezmoidata/packages.yaml` | `.brew_casks` | `.Values.packages.brew.casks` |
| `.chezmoidata/packages.yaml` | `.brew_taps` | `.Values.packages.brew.taps` |
| `.chezmoidata/packages.yaml` | `.go_packages` | `.Values.packages.go` |
| `.chezmoidata/packages.yaml` | `.npm_packages` | `.Values.packages.npm` |
| `.chezmoidata/mcp_servers.yaml` | `.mcp_servers` | `.Values.mcp_servers` |
| `.chezmoidata/mcp_permissions.yaml` | `.mcp_permissions` | `.Values.mcp_permissions` |
| `.chezmoidata/commands.yaml` | `.commands` | `.Values.commands` |
| `.chezmoidata/agents.yaml` | `.agents` | `.Values.agents` |
| `.chezmoidata/skills.yaml` | `.skills` | `.Values.skills` |
| `.chezmoidata/completions.yaml` | `.completions` | `.Values.completions` |

### 5.2 Template Function Mapping

| Chezmoi Function | Sprig Equivalent | Example |
|------------------|------------------|---------|
| `{{ env "VAR" }}` | `{{ .Env.VAR }}` | Direct mapping |
| `{{ .chezmoi.homeDir }}` | `{{ .Env.HOME }}` | Environment variable |
| `{{ range }}` | `{{ range }}` | Identical |
| `{{ if }}` | `{{ if }}` | Identical |
| `{{ has }}` | `{{ has }}` | Sprig function |
| `{{ hasKey }}` | `{{ hasKey }}` | Sprig function |
| `{{ keys }}` | `{{ keys }}` | Sprig function |
| `{{ sortAlpha }}` | `{{ sortAlpha }}` | Sprig function |
| `{{ index }}` | `{{ index }}` | Sprig function |
| `{{ join }}` | `{{ join }}` | Sprig function |
| `{{ lower }}` | `{{ lower }}` | Sprig function |
| `{{ replace }}` | `{{ replace }}` | Sprig function |
| `{{ sha256sum }}` | N/A | Not needed with state |

---

## 6. Testing Strategy

### 6.1 Pre-Migration Testing

```bash
# 1. Test values.yaml rendering
dotisan plan

# 2. Validate all templates render without errors
dotisan plan 2>&1 | grep -i error

# 3. Check specific resource
dotisan plan | grep -A 10 "ManagedFile"
```

### 6.2 Staged Migration Testing

**Phase 1 Test:**
```bash
# Apply only packages
dotisan apply --confirm

# Verify packages installed
brew list | grep starship
go list -m | grep gopls
npm list -g | grep task-master-ai

# Check state
dotisan state list
```

**Phase 2-6 Test:**
```bash
# Dry run each phase
dotisan plan

# Check file content before applying
dotisan plan | grep -A 5 "zshrc"

# Apply and verify
dotisan apply --confirm
cat ~/.zshrc | head -20
```

### 6.3 Post-Migration Validation

```bash
# Full state verification
dotisan state list

# Check for drift
dotisan plan
# Should show: "Plan: 0 to add, 0 to change, 0 to remove"

# Test removals work
dotisan state remove HomeBrewPackages core-tools
dotisan plan
# Should show removal plan
```

---

## 7. Rollback Plan

### 7.1 Before Migration

```bash
# Backup chezmoi state
cd ~/git/dotfiles
git stash
git checkout -b pre-dotisan-backup

# Export current chezmoi state
chezmoi state dump > ~/chezmoi-state-backup.json
```

### 7.2 During Migration

```bash
# After each phase, commit state
git add .
git commit -m "Phase X: Migrated [component]"

# Tag for rollback
git tag phase-X-complete
```

### 7.3 Rollback Procedure

```bash
# If issues detected:
# 1. Restore files from chezmoi
cd ~/git/dotfiles
chezmoi apply

# 2. Remove dotisan-managed resources (optional)
dotisan state list
dotisan state remove [kind] [name]

# 3. Switch back to chezmoi
# Update shell configs to use chezmoi paths
```

---

## 8. Migration Timeline

| Week | Phase | Components | Effort |
|------|-------|------------|--------|
| 1 | Foundation | Repository setup, values.yaml, packages | 4-6 hrs |
| 1-2 | Shell & Core | Zsh, Fish, Git configs | 3-4 hrs |
| 2 | Terminals | WezTerm, Ghostty | 2-3 hrs |
| 2-3 | AI Tools | OpenCode, Claude, VS Code MCP | 6-8 hrs |
| 3 | Agents | Agent skills directory | 2-3 hrs |
| 3 | Commands | Generated commands | 4-6 hrs |
| **Total** | | | **21-30 hrs** |

---

## 9. Success Criteria

- [ ] All 39 brew formulas managed by dotisan
- [ ] All 10 brew casks managed by dotisan
- [ ] All 7 Go packages managed by dotisan
- [ ] All 3 NPM packages managed by dotisan
- [ ] Zsh configuration migrated with templates working
- [ ] Fish configuration migrated
- [ ] All 3 MCP client configs generated from single source
- [ ] 14 AI agent skills in managed directory
- [ ] All static configs migrated
- [ ] `dotisan plan` shows 0 changes after migration
- [ ] Removal testing successful (can remove and re-add)
- [ ] State tracking working for all resources

---

## 10. Post-Migration Cleanup

After successful migration:

1. **Archive chezmoi repository:**
   ```bash
   mv ~/git/dotfiles ~/git/dotfiles-chezmoi-archived
   ```

2. **Update shell RC files:**
   - Remove chezmoi-specific paths
   - Add dotisan to PATH if needed

3. **Update documentation:**
   - Add migration notes to README
   - Document new dotisan workflow

4. **Clean up old state:**
   ```bash
   rm -rf ~/.local/share/chezmoi
   rm -rf ~/.cache/chezmoi
   ```

---

## Appendix A: File Mapping Table

| Source (Chezmoi) | Destination (Dotisan) | Resource Kind |
|------------------|----------------------|---------------|
| `.chezmoidata/packages.yaml` | `values.yaml` (merged) | Data |
| `.chezmoidata/mcp_servers.yaml` | `values.yaml` (merged) | Data |
| `.chezmoidata/mcp_permissions.yaml` | `values.yaml` (merged) | Data |
| `.chezmoidata/commands.yaml` | `values.yaml` (merged) | Data |
| `.chezmoidata/agents.yaml` | `values.yaml` (merged) | Data |
| `.chezmoidata/skills.yaml` | `values.yaml` (merged) | Data |
| `.chezmoidata/completions.yaml` | `values.yaml` (merged) | Data |
| `dot_zshrc.tmpl` | `resources/10-shell/zshrc.sh` + `zsh.yaml` | ManagedFile |
| `private_dot_config/fish/config.fish` | `resources/10-shell/fish.yaml` | ManagedFile |
| `dot_gitconfig.tmpl` | `resources/30-tools/git.yaml` | ManagedFile |
| `dot_ssh/config.tmpl` | `resources/30-tools/ssh.yaml` | ManagedFile |
| `private_dot_config/wezterm/wezterm.lua` | `resources/20-terminals/wezterm.yaml` | ManagedFile |
| `Library/Application Support/com.mitchellh.ghostty/config` | `resources/20-terminals/ghostty.yaml` | ManagedFile |
| `private_dot_config/starship/starship.toml` | `resources/30-tools/starship.yaml` | ManagedFile |
| `private_dot_config/zellij/config.kdl.tmpl` | `resources/30-tools/zellij.yaml` | ManagedFile |
| `private_dot_config/opencode/opencode.json.tmpl` | `resources/40-ai-tools/opencode.yaml` | ManagedFile |
| `private_dot_config/opencode/AGENTS.md` | `resources/40-ai-tools/opencode-agents.yaml` | ManagedFile |
| `dot_claude/settings.json.tmpl` | `resources/40-ai-tools/claude-code.yaml` | ManagedFile |
| `dot_claude/CLAUDE.md` | `resources/40-ai-tools/claude-agents.yaml` | ManagedFile |
| `Library/Application Support/Code/User/mcp.json.tmpl` | `resources/40-ai-tools/vscode-mcp.yaml` | ManagedFile |
| `Library/Application Support/Code/User/settings.json.tmpl` | `resources/40-ai-tools/vscode-settings.yaml` | ManagedFile |
| `dot_taskmaster_config.json.tmpl` | `resources/40-ai-tools/taskmaster.yaml` | ManagedFile |
| `dot_agents/skills/*/` | `resources/50-agents/skills/` | ManagedDirectory |

---

## Appendix B: Environment Variables Required

These environment variables must be available during `dotisan apply`:

| Variable | Usage | Required For |
|----------|-------|--------------|
| `HOME` | Path resolution | All files |
| `USER` | User name | Some configs |
| `CONTEXT7_API_TOKEN` | MCP server headers | context7 MCP |
| `GITHUB_PERSONAL_ACCESS_TOKEN` | MCP server headers | github MCP |
| `KARAKEEP_API_KEY` | MCP server env | karakeep MCP |
| `GOOGLE_API_KEY` | MCP server env | taskmaster MCP |

---

**END OF PRD**
