package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/pterm/pterm"
	"github.com/wasilak/dotisan/pkg/ui"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/resource"
	"github.com/wasilak/dotisan/pkg/state"
	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Manage the persisted state file",
	Long:  "Manage state file entries: add, move, remove, and list managed resources.",
}

// stateImportCmd imports an existing resource into state
var stateImportCmd = &cobra.Command{
	Use:          "import KIND/GROUP[ITEM] [ACTUAL_VALUE]",
	SilenceUsage: true,
	Short:        "Import existing resource into state",
	Long: `import discovers an existing resource on your system and adds it to
the state file without making any changes to the system.

ACTUAL_VALUE is optional. When omitted, the item name from the ID is used.
It is only needed when the logical state name differs from the actual resource
on the system (e.g. a managed file whose path differs from its state name).

Examples:
  dotisan state import HomeBrewPackages/homebrew-packages[ripgrep]
  dotisan state import HomeBrewPackages/homebrew-packages/fd[fd]
  dotisan state import ManagedFile/dotfiles[zshrc] ~/.zshrc`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		actual := ""
		if len(args) == 2 {
			actual = args[1]
		}
		return runStateImport(cmd.Context(), args[0], actual)
	},
}

// parseID/parsing is handled centrally by pkg/resource.ParseResourceID

func runStateImport(ctx context.Context, id, actual string) error {
	// Parse canonical ResourceID
	rid, err := resource.ParseResourceID(id)
	if err != nil {
		return err
	}
	if rid.Item == "" {
		return fmt.Errorf("invalid ID format: %s (item name required, e.g. Kind/group[item])", id)
	}
	kind, group, item := rid.Kind, rid.Group, rid.Item

	actualValue := actual
	if actualValue == "" {
		actualValue = rid.Item
	}

	// Ensure providers are registered (idempotent)
	ensureProvidersRegistered()

	// Get provider by kind (registry maps kinds to providers)
	p, err := provider.GetByKind(rid.Kind)
	if err != nil {
		return fmt.Errorf("provider not found for kind %s: %w", rid.Kind, err)
	}

	// Check if provider is available
	available, msg := p.Available()
	if !available {
		return fmt.Errorf("provider %s is not available: %s", rid.Kind, msg)
	}

	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	// Use provider Import (group-level). Providers return discovered group
	// state or an error. For backward compatibility, when import fails we
	// fall back to creating a minimal ResourceState with the provided item
	// so the CLI can still record it in state. This is a transitional
	// compatibility behavior; callers should prefer explicit import handling.
	resourceState, err := p.Import(ctx, rid.Group)
	if err != nil {
		// Fall back to minimal state with the requested item only
		resourceState = provider.ResourceState{
			Kind:  rid.Kind,
			Group: rid.Group,
			Items: []resource.ItemState{{Name: actualValue, Status: "present"}},
		}
	} else {
		// Find the requested item in the discovered set and keep only that one.
		// Import discovers the whole installed set; we must not write unrelated
		// packages into state.
		var foundItem *resource.ItemState
		for i, it := range resourceState.Items {
			if it.Name == actualValue {
				foundItem = &resourceState.Items[i]
				break
			}
		}
		if foundItem == nil {
			resourceState.Items = []resource.ItemState{{Name: actualValue, Status: "present"}}
		} else {
			resourceState.Items = []resource.ItemState{*foundItem}
		}

		// Ensure kind/group are set consistently with the parsed ID
		resourceState.Kind = rid.Kind
		resourceState.Group = rid.Group
	}

	// Load current state
	cfg, cfgErr := config.LoadConfigFromDefaultPath()
	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		slog.Warn("failed to load config", "err", cfgErr)
	}
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		currentState = state.NewState()
	}

	// Check if resource group already exists
	if _, exists := currentState.GetResourceGroup(kind, group); exists {
		// Append to existing group
		existing, _ := currentState.GetResourceGroup(kind, group)
		existing.Items = append(existing.Items, resourceState.Items...)
		currentState.SetResourceGroup(existing)
	} else {
		currentState.SetResourceGroup(resourceState)
	}

	// Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("%s Successfully imported %s/%s[%s]\n", style.IconSuccess, kind, group, item)
	return nil
}

// ensureProvidersRegistered registers all providers
func ensureProvidersRegistered() {
	if _, err := provider.Get("file"); err != nil {
		provider.Register("file", providers.NewFileProvider(""), resource.KindManagedFile)
	}
	if _, err := provider.Get("homebrew"); err != nil {
		provider.Register("homebrew", providers.NewBrewProvider(), resource.KindHomeBrewPackages, resource.KindHomeBrewCasks, resource.KindHomeBrewTaps)
	}
	if _, err := provider.Get("npm"); err != nil {
		provider.Register("npm", providers.NewNpmProvider(), resource.KindNpmPackages)
	}
	if _, err := provider.Get("go"); err != nil {
		provider.Register("go", providers.NewGoProvider(), resource.KindGoPackages)
	}
	if _, err := provider.Get("cargo"); err != nil {
		provider.Register("cargo", providers.NewCargoProvider(), resource.KindCargoPackages)
	}
}

// stateMoveCmd moves an item between resource groups in state
var stateMvCmd = &cobra.Command{
	Use:          "move SOURCE DESTINATION",
	Aliases:      []string{"mv"},
	SilenceUsage: true,
	Short:        "Move an item between resource groups in state",
	Long: `mv moves an item from one resource group to another in state only.
The actual system resource is not modified.

Source and destination format: Kind/Group/Item or Kind/Group
If destination item name is not provided, the source item name is used.

The destination group must exist in the desired configuration.

Examples:
  dotisan state mv HomeBrewPackages/core-tools/ripgrep HomeBrewPackages/homebrew-packages/ripgrep
  dotisan state mv HomeBrewPackages/core-tools/ripgrep HomeBrewPackages/homebrew-packages/`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStateMv(cmd.Context(), args[0], args[1])
	},
}

func runStateMv(ctx context.Context, source, destination string) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}

	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run state mv
	result, err := eng.StateMv(ctx, engine.StateMvOptions{
		Source:      source,
		Destination: destination,
	})
	if err != nil {
		fmt.Println()
		fmt.Println(style.Error.Render("✖ Move failed"))
		fmt.Println()
		fmt.Printf("  %s\n", err)
		fmt.Println()
		return fmt.Errorf("state mv failed")
	}

	// Display result
	engine.DisplayStateMvResult(result)
	return nil
}

// stateRemoveCmd removes a resource from state
var stateRemoveCmd = &cobra.Command{
	Use:          "remove KIND/GROUP or KIND/GROUP[ITEM]",
	Aliases:      []string{"rm"},
	SilenceUsage: true,
	Short:        "Remove resource group or item from state",
	Long: `remove deletes a resource group or a single item from the state file
without affecting the actual system.

Examples:
  dotisan state remove HomeBrewPackages/homebrew-packages
  dotisan state remove HomeBrewPackages/homebrew-packages/fd[fd]`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		return runStateRemoveByID(cmd.Context(), id)
	},
}

var stateRemoveForce bool
var stateRemoveConfirm bool

func init() {
	stateRemoveCmd.Flags().BoolVarP(&stateRemoveForce, "force", "f", false, "Skip confirmation prompt")
	stateRemoveCmd.Flags().BoolVar(&stateRemoveConfirm, "confirm", false, "Skip confirmation and remove without prompting")
}

func runStateRemoveByID(ctx context.Context, id string) error {
	rid, err := resource.ParseResourceID(id)
	if err != nil {
		return err
	}
	kind, group, item := rid.Kind, rid.Group, rid.Item

	if !stateRemoveForce && !stateRemoveConfirm {
		title := ""
		if item == "" {
			title = fmt.Sprintf("Remove %s/%s from state?", kind, group)
		} else {
			title = fmt.Sprintf("Remove %s/%s/%s from state?", kind, group, item)
		}
		confirm, err := pterm.DefaultInteractiveConfirm.Show(title)
		if err != nil {
			return fmt.Errorf("confirmation prompt error: %w", err)
		}
		if !confirm {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Remove cancelled."))
			return nil
		}
	}

	cfg, cfgErr := config.LoadConfigFromDefaultPath()
	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		slog.Warn("failed to load config", "err", cfgErr)
	}
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	currentState, err := backend.Load(ctx)
	if err != nil {
		return fmt.Errorf("cannot load state: %w", err)
	}

	var removed bool
	if item == "" {
		removed = currentState.RemoveResourceGroup(kind, group)
	} else {
		removed = currentState.RemoveResourceItem(kind, group, item)
	}

	if !removed {
		if item == "" {
			fmt.Printf("%s Resource %s/%s not found in state\n", style.IconError, kind, group)
		} else {
			fmt.Printf("%s Resource %s/%s/%s not found in state\n", style.IconError, kind, group, item)
		}
		return nil
	}

	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if item == "" {
		fmt.Printf("%s Removed %s/%s from state\n", style.IconSuccess, kind, group)
	} else {
		fmt.Printf("%s Removed %s/%s/%s from state\n", style.IconSuccess, kind, group, item)
	}
	return nil
}

// stateListCmd lists all managed resources
var stateListCmd = &cobra.Command{
	Use:          "list",
	Aliases:      []string{"ls"},
	SilenceUsage: true,
	Short:        "List all managed resources",
	Long: `list displays all resources currently tracked in the state file
along with their status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStateList(cmd.Context())
	},
}

var stateOutputFlag string

func runStateList(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	cfg, cfgErr := config.LoadConfigFromDefaultPath()
	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		slog.Warn("failed to load config", "err", cfgErr)
	}
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	var currentState *state.State
	var loadErr error
	// Use provided context so signal cancellation propagates to spinner and backend.Load
	err := style.RunWithSpinner(ctx, "Loading state...", func(ctx context.Context) error {
		currentState, loadErr = backend.Load(ctx)
		return loadErr
	})
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No state file found. Run 'dotisan apply' first.")
			return nil
		}
		return fmt.Errorf("cannot load state: %w", err)
	}

	if len(currentState.Resources) == 0 {
		fmt.Println("No managed resources found.")
		return nil
	}

	// Determine output format (render table as pterm resource table)
	outputFormat := output.Format(stateOutputFlag)
	if outputFormat == "" {
		outputFormat = output.FormatPlain
	}

	switch outputFormat {
	case output.FormatTree:
		return displayStateTree(currentState)
	case output.FormatJSON:
		return displayStateJSON(currentState)
	default:
		return displayStateTable(currentState)
	}
}

func displayStateTree(currentState *state.State) error {
	var resources []diff.StateResource
	for _, res := range currentState.Resources {
		items := make([]string, 0, len(res.Items))
		for _, item := range res.Items {
			items = append(items, item.Name)
		}
		resources = append(resources, diff.StateResource{
			Kind:   res.Kind,
			Group:  res.Group,
			Items:  items,
			Status: "managed",
		})
	}

	treeFormatter := diff.NewTreeFormatter()
	if err := treeFormatter.FormatStateAsTree(resources); err != nil {
		fmt.Fprintf(os.Stderr, "tree render error: %v\n", err)
	}
	return nil
}

func displayStateJSON(currentState *state.State) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(currentState)
}

func displayStateTable(currentState *state.State) error {
	fmt.Println(style.Header.Render("Managed Resources"))
	fmt.Println()

	// Display with resource table for state

	// Convert currentState (typed) to []ui.ResourceRow explicitly to avoid reflection
	rows := make([]ui.ResourceRow, 0)
	for _, res := range currentState.Resources {
		kind := res.Kind
		group := res.Group
		for _, it := range res.Items {
			status := it.Status
			if status == "" {
				status = "sync"
			}
			id := fmt.Sprintf("%s/%s[%s]", kind, group, it.Name)
			rows = append(rows, ui.ResourceRow{
				Status: status,
				ID:     id,
				Kind:   kind,
				Group:  group,
				Name:   it.Name,
				Info:   it.Version,
			})
		}
	}
	if err := ui.RenderResourceTable(rows, true); err != nil {
		fmt.Fprintf(os.Stderr, "resource table error: %v\n", err)
	}

	totalItems := 0
	for _, res := range currentState.Resources {
		totalItems += len(res.Items)
	}
	fmt.Printf("\nTotal: %d resources across %d groups\n", totalItems, len(currentState.Resources))
	return nil
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateImportCmd)
	stateCmd.AddCommand(stateMvCmd)
	stateCmd.AddCommand(stateRemoveCmd)
	stateCmd.AddCommand(stateListCmd)
	stateListCmd.Flags().StringVarP(&stateOutputFlag, "output", "o", "", "Output format (plain, tree, json)")

	// Customize state command help to show aliases for subcommands inline.
	// Cobra's default help doesn't display subcommand aliases in the list.
	stateCmd.SetHelpTemplate(`{{with (or .Long .Short)}}{{.}}{{end}}

Usage:
  {{.CommandPath}} [command]

Available Commands:
{{- range .Commands}}
  {{rpad .Name 12}} {{.Short}}{{if .Aliases}} (aliases: {{range $i, $a := .Aliases}}{{if $i}}, {{end}}{{$a}}{{end}}){{end}}
{{- end}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`)
}
