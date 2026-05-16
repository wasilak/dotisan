# Nim

## What This Is

Nim is a declarative dotfiles and machine configuration manager for personal use. It reads YAML resource manifests, computes a diff against persisted state, and applies changes through typed providers — bringing Terraform's plan/apply workflow to dotfiles. The current milestone adds **namespace support**, enabling different sets of resources on different machines (e.g. work vs. personal).

## Core Value

A user can safely apply a different set of resources on each machine without duplicating manifests, by declaring a namespace on any resource and setting the active namespace via env var or CLI flag.

## Requirements

### Validated

- ✓ Declarative plan/apply model with diff preview — Phase 1
- ✓ Resource manifests as Kubernetes-style YAML (kind, metadata, spec) — Phase 1
- ✓ 8 built-in providers: ManagedFile, HomeBrewPackages, Casks, Taps, NpmPackages, GoPackages, CargoPackages, AISkill — Phase 1
- ✓ DAG dependency ordering via `metadata.dependsOn` — Phase 1
- ✓ Two-pass Go template rendering (values.yaml + env vars, Sprig functions) — Phase 1
- ✓ Local JSON and S3-compatible state backends — Phase 1
- ✓ `--target` flag with `/pattern/` regex support — Phase 1

### Active

- [ ] Namespace field on resource manifests (`metadata.namespace`) — single regex value matching one or more namespaces
- [ ] Active namespace resolved from `NIM_NAMESPACE` env var or `--namespace` CLI flag; defaults to `"default"` when unset
- [ ] Resources with no `metadata.namespace` field implicitly belong to the `"default"` namespace
- [ ] Resources whose `metadata.namespace` regex matches the active namespace string are included in plan/apply
- [ ] `{{ .Namespace }}` available as a Go template variable for conditional content within template files
- [ ] Namespace filtering applied before DAG construction and plan diffing

### Out of Scope

- Multiple simultaneous active namespaces — only one namespace is active per nim invocation; use regex on resource side to express multi-namespace membership
- Namespace-specific state backends — state is shared across namespaces; namespace is a filter, not a partition
- Hostname-based auto-detection — user controls the active namespace explicitly via env/flag, not via hostname matching (hostname regex was considered but rejected in favour of explicit control)
- GUI/TUI for managing namespaces — CLI only

## Context

**Existing codebase state:**
- Config loading and two-pass template rendering lives in `pkg/config/` — this is where `.Namespace` injection belongs
- Engine (`pkg/engine/plan.go`, `pkg/engine/apply.go`) loads all resources then diffs/applies — namespace filtering must happen before resources reach the engine
- Resource YAML is parsed into `resource.Resource` structs; `metadata` fields already include `name`, `dependsOn`, etc. — `namespace` extends this struct
- Template context is currently the rendered `values.yaml` map — `.Namespace` needs to be merged in at the same level

**Known concerns to keep in mind (not in scope for this milestone):**
- Non-atomic state writes in `pkg/state/local.go` (separate fix)
- AISkillProvider idempotency bug (separate fix)

## Constraints

- **Compatibility**: Resources with no `metadata.namespace` must continue to work exactly as today when no namespace is set — zero breaking changes for existing dotfiles configs
- **Tech stack**: Go 1.26, Cobra CLI, Sprig templates — no new dependencies preferred
- **Testing**: stdlib `testing` only; table-driven tests following existing patterns in `pkg/providers/file_test.go` and `pkg/resource/generator_test.go`
- **Architecture**: No `context.Background()` in `pkg/`; wrap all errors with `fmt.Errorf("...: %w", err)`

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| `metadata.namespace` as single regex field | Supports multi-namespace membership via `/(work\|personal)/` without YAML list complexity | — Pending |
| Active namespace defaults to `"default"` when unset | Backward compatible — existing resources (no namespace) implicitly become "default" and keep working | — Pending |
| ENV var `NIM_NAMESPACE` + `--namespace` flag | Standard 12-factor pattern; env for machine-level default, flag for ad-hoc override | — Pending |
| `.Namespace` in template context | Enables conditional content in dotfile templates without external tooling | — Pending |
| Hostname auto-detection excluded | Explicit control is predictable; hostname matching can be emulated via NIM_NAMESPACE set in shell profile | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd:complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-16 after initialization*
