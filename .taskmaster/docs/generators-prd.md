Title: Generators PRD

Summary
- Implement a generator feature on ManagedFile resources that allows generating multiple files from a values list (.Values) using templates. The loader will expand generator specs into normal ManagedFile.Files in-memory so the rest of the system (engine, providers, state) treats them identically to statically-declared files.

Goals
- Allow maintaining lists-of-resources (e.g., skills, agents, snippets) in values and generate per-item files declaratively.
- Keep runtime/system behavior identical to regular ManagedFile resources after generation (no special-case handling required by providers or engine).
- Avoid introducing a new resource kind; use a `generator` field on ManagedFileSpec.
- Keep generation deterministic and testable.

Out Of Scope
- Runtime "clean" sweeps that remove unmanaged files outside of tracked destinations. The generator will not attempt to delete files it didn't generate and that aren't tracked in state.

Success Criteria
- Loader accepts a ManagedFile with a `generator` field and expands it into Files in-memory.
- Existing provider code requires no changes to handle generated files.
- Unit tests cover generator expansion, invalid configs, mutual exclusion with other source fields, and template rendering edge cases.

Assumptions
- The values tree (.Values) is available to the loader at generation time.
- Simple dot-notation is sufficient for sourceKey resolution (e.g., `skills`, `agents.skills`). Array indexing is not required for v1.
- Templates are Go-style text/templates.

Requirements
1. Data model
   - Add GeneratorSpec to ManagedFileSpec:
     - sourceKey (string, required): dot-notation path into .Values that resolves to a list/array.
     - template (string, required): Go text/template used to render file content for each item.
     - destinationPattern (string, required): Go text/template used to render each file's destination path.
     - mode (string, optional): file mode (e.g., "0644").

2. Validation
   - GeneratorSpec is mutually exclusive with Source/SourceFile/Files fields.
   - sourceKey must resolve to a list/array at load time; otherwise loader returns an error.
   - destinationPattern must render to an absolute or home-relative path; loader should expand ~ and validate the result.

3. Loader behavior
   - Resolve GeneratorSpec during resource load/parse phase (before plan/engine runs).
   - For each item in the resolved list:
     - Build a template context: { item: <value>, index: <int>, Values: <values>, Env: <env>, OS: <os> }
     - Render `template` to produce file content.
     - Render `destinationPattern` to compute destination path.
     - Create a FileSpec object for each rendered item with Source set to the rendered content and Destination set to the rendered path and Mode set to the provided mode (or default).
   - Replace the ManagedFile.Spec.Files with generated FileSpec entries in-memory and clear the `generator` field so downstream sees a normal ManagedFile.

4. Template context
   - .item — the current item from the list (can be scalar/struct/map)
   - .index — 0-based index in the list
   - .Values — full values tree available to the loader
   - .Env — environment variables available to the process
   - .OS — runtime OS info (e.g., GOOS)

5. Rendering
   - Use Go text/template for both content and destinationPattern.
   - Destination path templates must produce a string. If the result is not valid (empty or whitespace), loader errors.

6. State interactions
   - Generated files are standard ManagedFile.Files; state import/export, checksums, and removal behave as with normal files.
   - If a generated file is removed from state but the generator still produces it, it will reappear on the next plan/apply (desired state is authoritative).

7. CLI / UX
   - No CLI changes required for v1; usage documented in docs and examples.

8. Examples
   - Example ManagedFile with generator:

```yaml
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: claude-skills
spec:
  generator:
    sourceKey: "skills"
    template: |
      ---
      name: {{ .item.name }}
      description: {{ .item.description }}
      ---
      {{ .item.content }}
    destinationPattern: "~/.claude/skills/{{ .item.name }}.md"
    mode: "0644"
```

Design Details
- Mutual exclusion simplifies validation and avoids ambiguous inputs.
- Clearing the generator field after expansion keeps runtime types uniform and minimizes downstream code impact.
- Template context intentionally small for v1 — we can expand later if needed (e.g., helper funcs).

Loader Implementation Notes
- Added helper to resolve dot-notation sourceKey into values (only lists supported for now).
- Use existing template rendering utilities; add tests that assert deterministic rendering and correct context.
- Validate destination paths and normalize ~ to user home.

Validation & Tests
- Unit tests for:
  - Generator expands lists of simple scalars and maps to Files with correct destinations and content.
  - Invalid sourceKey (missing or not a list) results in loader error.
  - Mutual exclusion enforcement triggers validation error when both generator and Files/source are provided.
  - Template errors (syntax or runtime) are surfaced as loader errors with context (resource name, item index).

Migration Plan
- Convert any historical ManagedDirectory manifests into ManagedFile generator manifests or list-based `files:` entries (ManagedDirectory has been removed).
- Provide example migration snippet in migration docs showing replacement using `generator` or explicit `files:` listings.

Acceptance Criteria
- Loader expands generator specs into Files in-memory and the engine can plan/apply generated files with checksums persisted.
- No provider code changes beyond tests adjustments; existing file provider accepts generated Files transparently.
- Tests pass and new unit tests added.

Risks
- Template injection/escaping issues — mitigate by documenting that templates are raw text/template and advising proper escaping for filenames.
- Large lists may cause memory/plan size concerns — document expected limits and consider streaming generation in future.

Rollout Plan
1. Implement GeneratorSpec and loader expansion.
2. Add unit tests and update existing tests that referenced ManagedDirectory. Replace such tests with scenarios using ManagedFile generators or explicit `files:` lists.
3. Add docs and migration examples.
4. Merge behind feature branch; run integration tests and smoke apply on a small example.

Open Questions
- Should destinationPattern support relative paths that are relative to a base directory field on ManagedFile? (defer to v2)
- Should we allow additional template functions (e.g., `slugify`) in v1? (prefer to add later)

Ownership
- Implementer: engineering
- Reviewer: code owner for pkg/resource and pkg/engine
