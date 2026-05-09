package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wasilak/nim/pkg/config"
	"github.com/wasilak/nim/pkg/resource"
)

// Auditor performs manifest audits under a dotfiles root
type Auditor struct {
	DotfilesRoot string
}

func NewAuditor(dotfilesRoot string) *Auditor {
	return &Auditor{DotfilesRoot: dotfilesRoot}
}

// Run executes the audit. It loads resources using the existing loader and
// performs structural and cross-resource checks.
func (a *Auditor) Run() (*AuditResult, error) {
	// Prepare result
	res := &AuditResult{
		Issues:          []ValidationIssue{},
		DependencyGraph: DependencyGraph{},
	}

	// Prepare template context and render values.yaml first (same as Loader)
	ctx := config.NewTemplateContext()
	engine := config.NewTemplateEngine(ctx)
	valuesPath := filepath.Join(a.DotfilesRoot, "values.yaml")
	_ = engine.LoadAndRenderValues(valuesPath)

	// Use the resource loader which handles shielding generator templates,
	// rendering, parsing, validation and generator expansion. We added
	// LoadResourcesWithFiles to capture non-fatal file errors.
	loader := resource.NewLoader(a.DotfilesRoot, ctx)
	resources, fileFor, loadErrs := loader.LoadResourcesWithFiles()

	// Collect load errors as issues. Downgrade to warnings because resource
	// loading can be sensitive to runtime template helpers; treat them as
	// warnings to avoid false positives until full runtime fidelity is used.
	for _, e := range loadErrs {
		res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityWarning, Category: "load", Message: e.Error()})
	}

	res.TotalFiles = len(resources) + len(loadErrs)
	res.ValidFiles = len(resources)

	// Duplicate detection (by name)
	nameCount := map[string]int{}
	for _, r := range resources {
		name := r.GetMetadata().Name
		nameCount[name]++
	}
	for name, c := range nameCount {
		if c > 1 {
			// find files for duplicates
			var dupFiles []string
			for _, r := range resources {
				if r.GetMetadata().Name == name {
					dupFiles = append(dupFiles, fileFor[name])
				}
			}
			res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "duplicate", Resource: name, Message: fmt.Sprintf("duplicate resource name %q", name), Suggestion: "Rename or remove the duplicate resource"})
		}
	}

	// Build dependency graph
	// nodes: resource names; edges: dependsOn entries
	nodes := []ResourceNode{}
	edges := []DependencyEdge{}
	nameExists := map[string]bool{}
	for _, r := range resources {
		name := r.GetMetadata().Name
		nodes = append(nodes, ResourceNode{ID: name, Kind: r.GetKind(), File: fileFor[name], Status: "valid"})
		nameExists[name] = true
	}
	for _, r := range resources {
		from := r.GetMetadata().Name
		for _, dep := range r.GetMetadata().DependsOn {
			// ParseResourceID allowed multiple forms; in our simplified world
			// names are plain names; if user supplied Kind/... we extract last part.
			target := dep
			if strings.Contains(dep, "/") {
				parts := strings.Split(dep, "/")
				target = parts[len(parts)-1]
			}
			exists := nameExists[target]
			edges = append(edges, DependencyEdge{From: from, To: target, Exists: exists, IsCyclic: false})
			if !exists {
				res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityWarning, Category: "dependency", File: fileFor[from], Resource: from, Message: fmt.Sprintf("dependency target %q not found", target), Suggestion: "Ensure the target resource exists or remove the dependency"})
			}
		}
	}

	// Detect cycles using DFS on adjacency map
	adj := map[string][]string{}
	for _, e := range edges {
		if e.Exists {
			adj[e.From] = append(adj[e.From], e.To)
		}
	}
	visited := map[string]int{} // 0=unseen,1=visiting,2=done
	var cycleFound bool
	var dfs func(string) bool
	dfs = func(n string) bool {
		if visited[n] == 1 {
			return true
		}
		if visited[n] == 2 {
			return false
		}
		visited[n] = 1
		for _, nb := range adj[n] {
			if dfs(nb) {
				return true
			}
		}
		visited[n] = 2
		return false
	}
	for _, node := range nodes {
		if visited[node.ID] == 0 {
			if dfs(node.ID) {
				cycleFound = true
				break
			}
		}
	}
	if cycleFound {
		res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "dependency", Message: "circular dependency detected", Suggestion: "Break the cycle between resources"})
	}

	// Mark cyclic edges if cycle found (simple conservative mark: if visited was used)
	// (we already recorded an error)

	// File references validation for ManagedFile
	for _, r := range resources {
		if r.GetKind() == resource.KindManagedFile {
			mf, ok := r.(*resource.ManagedFile)
			if !ok {
				continue
			}
			// Check single-file sourceFile
			if mf.Spec.SourceFile != "" {
				path := mf.Spec.SourceFile
				if !filepath.IsAbs(path) {
					// relative to resource file
					base := filepath.Dir(fileFor[mf.GetMetadata().Name])
					path = filepath.Join(base, path)
				}
				if _, err := os.Stat(path); err != nil {
					res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "file", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("sourceFile %s does not exist", mf.Spec.SourceFile)})
				}
			}
			// Check files list
			for _, f := range mf.Spec.Files {
				if f.SourceFile != "" {
					path := f.SourceFile
					if !filepath.IsAbs(path) {
						base := filepath.Dir(fileFor[mf.GetMetadata().Name])
						path = filepath.Join(base, path)
					}
					if _, err := os.Stat(path); err != nil {
						res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "file", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("sourceFile %s does not exist", f.SourceFile)})
					}
				}
			}
		}
	}

	// Template syntax validation: check ManagedFile.Template and generator templates
	// (we only check that Template strings parse as Go templates)
	for _, r := range resources {
		if r.GetKind() == resource.KindManagedFile {
			mf := r.(*resource.ManagedFile)
			if mf.Spec.Template {
				// If SourceFile is template, check its syntax
				if mf.Spec.SourceFile != "" {
					base := filepath.Dir(fileFor[mf.GetMetadata().Name])
					path := mf.Spec.SourceFile
					if !filepath.IsAbs(path) {
						path = filepath.Join(base, path)
					}
					data, err := os.ReadFile(path)
					if err == nil {
						// parse
						if _, err := parseTemplateString(string(data)); err != nil {
							res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "template", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("template parse error in %s: %v", mf.Spec.SourceFile, err)})
						}
					}
				}
			}
			for _, fs := range mf.Spec.Files {
				if fs.Template && fs.SourceFile != "" {
					base := filepath.Dir(fileFor[mf.GetMetadata().Name])
					path := fs.SourceFile
					if !filepath.IsAbs(path) {
						path = filepath.Join(base, path)
					}
					data, err := os.ReadFile(path)
					if err == nil {
						if _, err := parseTemplateString(string(data)); err != nil {
							res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "template", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("template parse error in %s: %v", fs.SourceFile, err)})
						}
					}
				}
			}
			if mf.Spec.Generator != nil {
				// check Template and SourceFilePattern fields as templates
				if mf.Spec.Generator.Template != "" {
					if _, err := parseTemplateString(mf.Spec.Generator.Template); err != nil {
						res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "template", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("generator template parse error: %v", err)})
					}
				}
				if mf.Spec.Generator.SourceFilePattern != "" {
					if _, err := parseTemplateString(mf.Spec.Generator.SourceFilePattern); err != nil {
						res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "template", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("generator sourceFilePattern parse error: %v", err)})
					}
				}
				if mf.Spec.Generator.DestinationPattern != "" {
					if _, err := parseTemplateString(mf.Spec.Generator.DestinationPattern); err != nil {
						res.Issues = append(res.Issues, ValidationIssue{Severity: SeverityError, Category: "template", File: fileFor[mf.GetMetadata().Name], Resource: mf.GetMetadata().Name, Message: fmt.Sprintf("generator destinationPattern parse error: %v", err)})
					}
				}
			}
		}
	}

	// Build dependency graph result
	res.DependencyGraph = DependencyGraph{Nodes: nodes, Edges: edges}
	// Build summary
	var errs, warns int
	for _, it := range res.Issues {
		if it.Severity == SeverityError {
			errs++
		} else if it.Severity == SeverityWarning {
			warns++
		}
	}
	res.Summary = AuditSummary{TotalResources: len(nodes), Valid: len(nodes) - errs - warns, Errors: errs, Warnings: warns}

	return res, nil
}

// parseTemplateString is implemented in template_parse.go
