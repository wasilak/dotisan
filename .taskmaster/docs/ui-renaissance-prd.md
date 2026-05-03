# dotisan UI Renaissance PRD
# ===========================

## 1. Overview

**Project Name:** dotisan UI Renaissance
**Status:** Proposed
**Priority:** High
**Target:** 2026 Q2

### Executive Summary

The dotisan CLI currently delivers a fragmented visual experience. While the `plan` command uses beautiful Charmbracelet lipgloss styling, other commands remain utilitarian plain-text output. This project transforms dotisan into a visually stunning, modern CLI experience that delights users and rival industry-leading tools like GitHub CLI (`gh`), Vercel CLI, and Warp.

### Current State Analysis

**What's Excellent:**
- `plan` command uses beautiful pastel palette (mint green, salmon, cream yellow, peach)
- Nice icons: ✚ ✖ ✎ ⚠ ✓ consistently used in plan
- `PlanFormatter` provides structured styling system
- lipgloss imported throughout codebase (121 references)

**What's Broken:**
- Only `plan` uses rich styling - everything else is plain text
- No inline syntax highlighting for resource IDs within sentences
- Basic printf-style tables with no borders or colors
- No visual hierarchy - output is a flat wall of text
- Commands feel disconnected from each other
- No progress indicators for multi-step operations
- No interactive elements beyond basic prompts
- Error messages lack visual emphasis

---

## 2. Problem Statement

### User Pain Points

1. **Inconsistent Experience**: `plan` is beautiful, `state list` is 2001-era terminal
2. **Poor Scanability**: Cannot quickly find important information in output
3. **No Emotional Design**: Purely functional, never delightful
4. **Verbose Output Fatigue**: Walls of text without visual breaks
5. **No Context**: Resource IDs stand alone without visual connection to surrounding text

### Business Impact

- Users perceive project as "dead" or abandoned
- Reduced daily usage motivation
- Poor impression for new users
- Missed opportunity for developer delight

---

## 3. Proposed Solution

### Design Principles

1. **Consistency Everywhere**: Every command uses identical style system
2. **Semantic Highlighting**: Resource types, paths, keywords all colored differently within sentences
3. **Visual Hierarchy**: Clear section breaks, boxed content, breadcrumbs
4. **Performance**: Styling adds <5ms overhead
5. **Progressive Enhancement**: Graceful degradation in limited terminals

### Color Palette

```go
// Core semantic colors
const (
    ColorMint      = "114"  // Success, additions, primary actions
    ColorCoral     = "174"  // Errors, deletions, destructive actions
    ColorCream     = "222"  // Info, modifications, secondary text
    ColorPeach     = "216"  // Warnings, drift detection
    ColorSlate     = "240"  // Dimmed, muted text, in-sync

    // Extended semantic colors
    ColorBlue      = "75"   // Links, interactive elements
    ColorPurple    = "141"  // Resource IDs (primary focus)
    ColorCyan      = "45"   // File paths, directories
    ColorPink      = "219"  // Package names, brew packages
    ColorGold      = "220"  // Highlights, current item
)
```

### Iconography (Expanded)

```go
const (
    IconAdd        = "✚"
    IconRemove     = "✖"
    IconEdit       = "✎"
    IconWarn       = "⚠"
    IconOK         = "✓"
    IconInfo       = "ℹ"
    IconLoading    = "◐"
    IconArrowRight = "→"
    IconArrowDown  = "↓"
    IconDot        = "•"
    IconSparkle    = "✨"
    IconRocket     = "🚀"
    IconCheck      = "✔"
    IconCross      = "✘"
    IconQMark      = "❓"
)
```

---

## 4. Technical Specification

### 4.1 Enhanced Style System

#### New File: `pkg/style/styles.go`

**Complete rewrite with:**

1. **Semantic Color System**
   - `StyleResourceID` - Purple for resource IDs like `ManagedFile/zshrc`
   - `StylePath` - Cyan for file paths like `~/.zshrc`
   - `StylePackageName` - Pink for package names like `ripgrep`
   - `StyleKeyword` - Cream for keywords like `in_sync`, `drifted`
   - `StyleLink` - Blue for URLs and references
   - `StyleHighlight` - Gold for emphasis

2. **Icon System**
   - All icons pre-styled with semantic colors
   - Animation frames for spinners
   - Category icons (providers, resources, state)

3. **Layout Primitives**
   - `Section(title string, content string) string`
   - `Card(icon, title, description string) string`
   - `ProgressBar(current, total int, label string) string`
   - `Table(headers []string, rows [][]string) string`
   - `Box(content string, borderStyle BorderStyle) string`
   - `Breadcrumb(items []string) string`
   - `Collapsible(header, content string) string`

4. **Typography**
   - Monospace numbers for counts
   - Proper unicode width handling
   - Truncation with "..." indicator

### 4.2 Command Enhancements (Comprehensive)

#### A. state list - Full Table Makeover

**Current:**
```
KIND                 NAME                      ID                                  STATUS
-----------------------------------------------------------------------------------------
ManagedFile          zshrc                     ManagedFile/zshrc                   in_sync
```

**Proposed:**
```
╭─────────────────────────────────────────────────────────────────────────────────────╮
│  💾 Managed Resources                                                          │
├─────────────────────────────────────────────────────────────────────────────────────┤
│  KIND          │ NAME       │ ID                               │ STATUS           │
├───────────────┼────────────┼──────────────────────────────────┼───────────────────┤
│  🔵 ManagedFile│ zshrc      │ 🔴 ManagedFile/zshrc             │ ✓ in_sync        │
│  🔵 ManagedFile│ vimrc      │ 🔴 ManagedFile/vimrc             │ ⚠ drifted       │
│  🟣 BrewPackages│ core-tools │ 🟣 BrewPackages/core-tools[rip..]│ ⚠ missing      │
├───────────────┴────────────┴──────────────────────────────────┴───────────────────┤
│  📊 Total: 3 resources  │  ✓ 1 in sync  │  ⚠ 2 need attention                │
╰─────────────────────────────────────────────────────────────────────────────────────╯
```

**Features:**
- Unicode box-drawing characters for borders
- Semantic icons per resource type
- Resource ID column with `StyleResourceID` color
- Status column color-coded (green/orange/red)
- Summary footer with counts
- Sortable (future: click to sort)
- Filter indicator in corner

#### B. state import - Rich Feedback

**Current:**
```
✓ Imported BrewPackages/core-tools[ripgrep]
Error: resource already exists
```

**Proposed:**
```
╭─────────────────── Import Result ───────────────────╮
│                                                       │
│  ✨ Successfully imported resource                   │
│                                                       │
│  Resource:  🔴 BrewPackages/core-tools[ripgrep]     │
│  Provider:  🟣 homebrew                            │
│  Status:    ✓ Added to state                       │
│                                                       │
╰──────────────────────────────────────────────────────╯
```
OR on error:
```
╭─────────────────── Import Error ────────────────────╮
│                                                       │
│  ✘ Cannot import: resource already exists            │
│                                                       │
│  Resource:  🔴 BrewPackages/core-tools[ripgrep]     │
│  Existing:  State already tracks this resource       │
│                                                       │
│  💡 Suggestion: Remove first with:                   │
│     dotisan state remove BrewPackages/core-tools[..]│
│                                                       │
╰──────────────────────────────────────────────────────╯
```

**Features:**
- Boxed output with icons
- Inline syntax highlighting for resource IDs
- Actionable error suggestions
- Provider type icon

#### C. apply - Visual Progress

**Current:**
```
Applying changes...
Created ManagedFile/zshrc
Created BrewPackages/ripgrep
Apply complete! 2 resources created.
```

**Proposed:**
```
╭─────────────────── Applying Changes ───────────────────╮
│                                                          │
│  🚀 Synchronizing your dotfiles...                       │
│                                                          │
│  Progress: [████████████████░░░░░░░░] 40%             │
│  ETA: ~5 seconds                                         │
│                                                          │
│  ─────────────────────────────────────────────────────  │
│                                                          │
│  ✓ ManagedFile/zshrc              created             │
│  ✓ ManagedFile/vimrc               created             │
│  ✓ BrewPackages/ripgrep            installed            │
│  ○ BrewPackages/fd                 pending              │
│                                                          │
╰──────────────────────────────────────────────────────────╯

╭─────────────────── Apply Complete! ────────────────────╮
│                                                          │
│  🎉 All resources synchronized!                         │
│                                                          │
│  Summary:                                               │
│    ✓ 3 created                                          │
│    ✎ 1 updated                                          │
│    ✖ 0 removed                                          │
│                                                          │
│  ⏱️ Completed in 8.2s                                   │
│                                                          │
╰──────────────────────────────────────────────────────────╯
```

**Features:**
- Progress bar with percentage
- ETA estimation
- Individual resource status with icons
- Animated spinner during wait
- Summary card with counts
- Timing information

#### D. init - Welcome Experience

**Current:**
```
Created ~/.config/dotisan/config.yaml
Created ~/.config/dotisan/values.yaml
Created directory ~/.config/dotisan/resources
```

**Proposed:**
```
╭──────────────────────────────────────────────────────────────────────────╮
│                                                                          │
│     █████╗  ██████╗ ██████╗███████╗███████╗███╗   ███╗                  │
│    ██╔══██╗██╔════╝██╔════╝██╔════╝██╔════╝████╗ ████║                  │
│    ███████║██║     ██║     █████╗  ███████╗██╔████╔██║                  │
│    ██╔══██║██║     ██║     ██╔══╝  ╚════██║██║╚██╔╝██║                  │
│    ██║  ██║╚██████╗╚██████╗███████╗███████║██║ ╚═╝ ██║                  │
│    ╚═╝  ╚═╝ ╚═════╝ ╚═════╝╚══════╝╚══════╝╚═╝     ╚═╝                  │
│                                                                          │
│                    dotisan v0.1.0                                        │
│               Your dotfiles management solution                          │
│                                                                          │
╰──────────────────────────────────────────────────────────────────────────╯

  ⚡ Setting up your environment...

  ✓ Created config directory
  ✓ Created config.yaml
  ✓ Created values.yaml
  ✓ Created resources/

╭──────────────────────────────────────────────────────────────────────────╮
│  📋 Next Steps                                                          │
│                                                                          │
│  1. 🔵 Edit    ~/.config/dotisan/resources/your-resource.yaml          │
│  2. 🟢 Run     dotisan plan                                           │
│  3. 🟢 Run     dotisan apply                                           │
│                                                                          │
╰──────────────────────────────────────────────────────────────────────────╯
```

**Features:**
- ASCII art logo with color gradient (if terminal supports)
- Step-by-step progress indicators
- Visual "cards" for next steps
- Semantic icons for actions

#### E. doctor - Diagnostic Dashboard

**Current:**
```
Checking dotisan installation...

✓ homebrew provider: available
✓ file provider: available
✗ npm provider: not found
✓ go provider: available
✓ cargo provider: available
```

**Proposed:**
```
╭───────────────────────────────── dotisan doctor ─────────────────────────────────╮
│                                                                                 │
│  🔍 Running Diagnostics...                                                      │
│                                                                                 │
├────────────────────────────────── Core ────────────────────────────────────────┤
│                                                                                 │
│  ✓ Config              Valid                    config.yaml loaded              │
│  ✓ State Backend      Ready                    local backend                  │
│  ✓ Templates          3 loaded                  All render correctly           │
│                                                                                 │
├──────────────────────────────── Providers ───────────────────────────────────────┤
│                                                                                 │
│  ✓ homebrew          Available                  v5.2.0                        │
│  ✓ file              Available                  Native                        │
│  ✘ npm               Not installed              Run: brew install npm          │
│  ✓ go                Available                  v1.24.0                        │
│  ✓ cargo             Available                  v0.16.0                        │
│                                                                                 │
├──────────────────────────────── Issues ─────────────────────────────────────────┤
│                                                                                 │
│  ⚠ 1 warning found                                                             │
│     • npm provider not installed - some features unavailable                  │
│                                                                                 │
╰────────────────────────────────────────────────────────────────────────────────╯
```

**Features:**
- Grouped sections (Core, Providers, Issues)
- Status icons per item
- Version information
- Actionable suggestions inline
- Issue count badge

#### F. plan - Enhanced Output

**Add to existing beautiful output:**
- Collapsible diff sections (expand/collapse)
- Warning cards with suggestions
- Resource type icons
- Progress for large plans
- Breadcrumb for multi-file changes

### 4.3 Implementation Components

#### Component 1: Style System (`pkg/style/styles.go`)

```go
package style

// Semantic color constants
const (
    ColorResourceID = "141"  // Purple - ManagedFile/zshrc
    ColorPath       = "45"    // Cyan - ~/.zshrc
    ColorPackage    = "219"   // Pink - ripgrep
    ColorKeyword    = "222"   // Cream - in_sync
    ColorLink       = "75"    // Blue - URLs
    ColorHighlight  = "220"   // Gold - emphasis
)

// Pre-styled render functions
func ResourceID(id string) string   { return ResourceIDStyle.Render(id) }
func Path(p string) string          { return PathStyle.Render(p) }
func Package(p string) string       { return PackageStyle.Render(p) }
func Keyword(k string) string       { return KeywordStyle.Render(k) }

// Layout primitives
func Section(title, content string) string
func Card(icon, title, desc string) string
func ProgressBar(current, total int) string
func Table(headers []string, rows [][]string) string
func Box(content string) string
func Breadcrumb(items []string) string
func ErrorBox(title, message, suggestion string) string
func SuccessBox(title, message string) string
```

#### Component 2: Interactive Prompts (`pkg/ui/prompts.go`)

```go
package ui

// Confirmation with styled prompt
func Confirm(prompt string) (bool, error)

// Select from options
func Select(options []string, label string) (int, error)

// Multi-select for bulk operations
func MultiSelect(options []string, label string) ([]int, error)

// Loading spinner with message
func Spinner(message string) *Spinner

// Progress tracker
func Progress(label string, total int) *Progress
```

#### Component 3: Table Renderer (`pkg/ui/table.go`)

```go
package ui

type Table struct {
    Headers   []string
    Rows      [][]string
    Align     []Align // Left, Center, Right
    Styles    TableStyles
}

func (t *Table) Render() string
func (t *Table) WithBorder(border BorderStyle) *Table
func (t *Table) WithRowColors(colors []lipgloss.Color) *Table
func (t *Table) SortBy(column int, asc bool) *Table
```

#### Component 4: Markdown Renderer (`pkg/ui/markdown.go`)

```go
package ui

// Render markdown-like syntax
func Render(s string) string
// Supports:
// - **bold** -> bold styling
// - `code` -> code styling
// - [link](url) -> link styling
// - # Header -> header styling
```

---

## 5. Implementation Phases

### Phase 1: Foundation (Week 1)

**Goal:** Core style system and layout primitives

- [ ] Enhance `pkg/style/styles.go` with semantic colors
- [ ] Add layout primitives (Section, Card, Box, Table)
- [ ] Add table renderer with borders and alignment
- [ ] Add breadcrumb component
- [ ] Add error/success box components

**Deliverables:**
- `pkg/style/styles.go` - Complete style system
- `pkg/ui/` package - Layout components

### Phase 2: Core Commands (Week 2-3)

**Goal:** Highest-impact commands fully modernized

- [ ] `state list` - Full table makeover with borders, icons, semantic colors
- [ ] `state import` - Rich success/error boxes with suggestions
- [ ] `state remove` - Confirmation prompt redesign
- [ ] `apply` - Progress bar and status cards
- [ ] Error handling - Consistent error presentation

**Deliverables:**
- All state subcommands modernized
- Apply command with progress
- Consistent error handling

### Phase 3: Enhancement (Week 3-4)

**Goal:** Polish remaining commands and add delight

- [ ] `init` - ASCII logo and welcome experience
- [ ] `doctor` - Diagnostic dashboard layout
- [ ] `plan` - Enhancements (collapsible diffs, warnings)
- [ ] Interactive prompts for confirmations
- [ ] Add spinners for async operations

**Deliverables:**
- Complete command modernization
- Interactive elements
- Delightful micro-interactions

### Phase 4: Advanced (Week 4+)

**Goal:** Nice-to-have features

- [ ] Sortable tables (click to sort)
- [ ] Filter/search in tables
- [ ] Sound effects (optional)
- [ ] Animation frames for spinners
- [ ] Terminal size responsiveness
- [ ] Theme customization

---

## 6. Acceptance Criteria

### Must Have (Phase 1-2)

- [ ] All commands use semantic color system
- [ ] Resource IDs always colored purple within sentences
- [ ] File paths always colored cyan
- [ ] `state list` renders as bordered table with icons
- [ ] `state import` shows boxed output with suggestions
- [ ] `apply` shows progress indicator
- [ ] Consistent error presentation everywhere

### Should Have (Phase 3)

- [ ] `init` shows ASCII logo
- [ ] `doctor` shows grouped dashboard
- [ ] Confirmation prompts are styled
- [ ] All icons are semantic (provider type icons)

### Nice to Have (Phase 4)

- [ ] Interactive table sorting
- [ ] Filter in tables
- [ ] Animations
- [ ] Theme support

---

## 7. Visual Checkpoints

1. Running any command feels like using the same polished app
2. Resource IDs are immediately recognizable (purple)
3. File paths stand out (cyan)
4. Errors are clearly boxed with actionable suggestions
5. Progress is visible during long operations
6. Tables are beautiful with proper alignment
7. CLI feels modern, alive, and delightful

---

## 8. Dependencies

### External
- `github.com/charmbracelet/lipgloss` - Already in use
- `github.com/charmbracelet/lipgloss-table` - For table rendering (check if needed)
- No new dependencies required

### Internal
- `pkg/style/styles.go` - Enhanced style system
- `pkg/ui/` - New UI components package
- Updates to all `cmd/*.go` files

---

## 9. Success Metrics

### Quantitative
- All 8 CLI commands use shared style system
- Zero plain text error messages
- <10ms overhead per command
- Semantic highlighting in 100% of applicable outputs

### Qualitative
- Users describe CLI as "beautiful", "modern", "delightful"
- No complaints about inconsistent output
- Social media mentions of CLI beauty (target: 0 → 5 in quarter)

---

## 10. Non-Functional Requirements

- Performance: Style rendering <10ms per command
- Compatibility: Works in terminals 80x24 and above
- Accessibility: Icons provide meaning beyond color
- Maintainability: Single source of truth for all styles

---

**PRD Created:** 2026-04-21
**Author:** Claude (AI Assistant)
**Status:** Ready for Review → Implementation

---

## Appendix A: Resource Kind Icons

```go
const (
    IconManagedFile      = "🔵"  // Blue circle
    // IconManagedDirectory removed: ManagedDirectory resource kind no longer exists
    IconBrewPackage      = "🟣"  // Purple circle
    IconNpmPackage      = "🟢"  // Green circle
    IconGoModule        = "🔷"  // Blue diamond
    IconCargoCrate      = "🟠"  // Orange circle
    IconGeneric         = "⚪"  // White circle
)
```

## Appendix B: Status Icons

```go
const (
    IconInSync   = "✓"  // Green check
    IconDrifted  = "⚠"  // Orange warning
    IconModified = "✎"  // Yellow pencil
    IconNew      = "✚"  // Green plus
    IconRemoved  = "✖"  // Red X
    IconPending  = "○"  // Hollow circle
    IconUnknown  = "?"  // Question mark
)
```

## Appendix C: Example Output - Full state list

```
╭──────────────────────────────────────────────────────────────────────────────────────╮
│  💾 dotisan state                                                                  │
├──────────────────────────────────────────────────────────────────────────────────────┤
│  🔍 Filter: [________________]  ℹ️ Press Enter to search                            │
├───────────────────────┬─────────────────────────────────────────────────────────────┤
│  🔵 ManagedFile      │ vimrc                         │ ~/.vimrc                   │
│  🔵 ManagedFile      │ zshrc                        │ ~/.zshrc                   │
│  📁 ManagedDirectory│ dotfiles                     │ ~/dotfiles                 │
│  🟣 BrewPackages    │ cli-tools    [ripgrep, fzf]  │ 2 packages                 │
├───────────────────────┴─────────────────────────────────────────────────────────────┤
│  📊 4 resources  │  ✓ 2 in sync  │  ⚠ 1 drifted  │  ✎ 1 modified                │
╰──────────────────────────────────────────────────────────────────────────────────────╯
```
