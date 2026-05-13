package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/wasilak/nim/pkg/cmdutil"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
)

// AISkillProvider installs AI skill packages from GitHub using the `skills` CLI.
type AISkillProvider struct{}

// NewAISkillProvider creates a new AISkillProvider.
func NewAISkillProvider() *AISkillProvider {
	return &AISkillProvider{}
}

// Name returns the provider name.
func (p *AISkillProvider) Name() string {
	return "aiskill"
}

// Available checks if npx is available on this system.
func (p *AISkillProvider) Available() (bool, string) {
	if path := cmdutil.CheckExecutable("npx"); path == "" {
		return false, "npx not found in PATH; install Node.js from https://nodejs.org/"
	}
	return true, "npx found"
}

// Reconcile compares the desired resource groups with the current system state.
// State is tracked via nim's state backend since the skills CLI does not
// expose which source repo each installed skill originated from.
func (p *AISkillProvider) Reconcile(
	ctx context.Context,
	desired []resource.ResourceGroup[any],
	state []provider.ResourceState,
) provider.GroupPlan {
	return provider.BaseReconcile(resource.KindAISkillPackages, desired, state, p.getInstalledSources(ctx), nil)
}

// getInstalledSources returns a map of source → version for globally installed skill packages.
// Since the skills CLI list output does not include source repos, we return an empty map
// and rely entirely on nim's state backend for tracking.
func (p *AISkillProvider) getInstalledSources(_ context.Context) map[string]string {
	return make(map[string]string)
}

// Apply executes the given GroupPlan.
func (p *AISkillProvider) Apply(ctx context.Context, plan provider.GroupPlan) ([]provider.ApplyItemResult, error) {
	var results []provider.ApplyItemResult
	for _, addition := range plan.Additions {
		results = append(results, p.applyGroupAddition(ctx, addition)...)
	}
	for _, removal := range plan.Removals {
		results = append(results, p.applyGroupRemoval(ctx, removal)...)
	}
	return results, nil
}

func (p *AISkillProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) []provider.ApplyItemResult {
	var results []provider.ApplyItemResult
	for _, item := range addition.Items {
		r := provider.ApplyItemResult{Kind: addition.Kind, Group: addition.Group, Item: item.Name, Op: "add"}
		args := []string{"--yes", "skills", "add", item.Name, "--global", "--yes"}
		if targets, ok := item.Metadata["targets"]; ok && targets != "" {
			for _, t := range strings.Split(targets, ",") {
				args = append(args, "--agent", t)
			}
		} else {
			args = append(args, "--all")
		}
		slog.Info("installing AI skill package", "source", item.Name)
		stdout, stderr, err := cmdutil.RunSimpleFn(ctx, "npx", args...)
		if err != nil {
			cmd := "npx " + strings.Join(args, " ")
			output := stderr
			if output == "" {
				output = stdout
			}
			r.Err = fmt.Errorf("failed to install %s: %s\n  command: %s", item.Name, output, cmd)
		}
		results = append(results, r)
	}
	return results
}

func (p *AISkillProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) []provider.ApplyItemResult {
	var results []provider.ApplyItemResult
	for _, item := range removal.Items {
		r := provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove"}
		skillNames, err := p.listSkillNames(ctx, item.Name)
		if err != nil {
			r.Err = fmt.Errorf("failed to list skills for %s: %w", item.Name, err)
			results = append(results, r)
			continue
		}
		if len(skillNames) == 0 {
			slog.Warn("no skills found to remove", "source", item.Name)
			results = append(results, r)
			continue
		}

		slog.Info("removing AI skill package", "source", item.Name, "skills", strings.Join(skillNames, ", "))
		args := []string{"--yes", "skills", "remove", "--global", "--yes", "--agent", "*", "--skill", strings.Join(skillNames, ",")}
		if _, stderr, err := cmdutil.RunSimpleFn(ctx, "npx", args...); err != nil {
			r.Err = fmt.Errorf("failed to remove skills from %s: %s: %w", item.Name, stderr, err)
		}
		results = append(results, r)
	}
	return results
}

// listSkillNames queries the skills CLI for the skill names provided by a source repo
// without installing them (uses --list flag).
func (p *AISkillProvider) listSkillNames(ctx context.Context, source string) ([]string, error) {
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "npx", "--yes", "skills", "add", source, "--list", "--json")
	if err != nil {
		slog.Warn("skills add --list failed", "source", source, "err", err)
		return nil, nil
	}

	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(stdout), &entries); err != nil {
		slog.Warn("skills add --list: failed to parse json", "source", source, "err", err)
		return nil, nil
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names, nil
}

// Import is not supported for the AI skill provider.
func (p *AISkillProvider) Import(_ context.Context, group string) (provider.ResourceState, error) {
	return provider.ResourceState{}, fmt.Errorf("import not supported for provider aiskill (group: %s)", group)
}
