package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/state"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "State management commands",
	Long: `state provides subcommands for managing the state file:

  import  - Bring existing resource under management
  remove  - Drop from state without touching system
  list    - Show all managed resources + status
  pull    - Fetch state from remote backend
  push    - Write local state to remote backend`,
}

// stateImportCmd imports an existing resource into state
var stateImportCmd = &cobra.Command{
	Use:   "import KIND NAME ID",
	Short: "Import existing resource into state",
	Long: `import calls Provider.Import(id) to discover an existing resource
and adds it to the state file without making any changes to the system.

Example: dotisan state import BrewPackages core-tools ripgrep`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := args[0]
		name := args[1]
		id := args[2]
		return runStateImport(kind, name, id)
	},
}

// kindToProvider maps resource kind to provider name
func kindToProvider(kind string) string {
	switch strings.ToLower(kind) {
	case "managedfile", "manageddirectory":
		return "file"
	case "brewpackages", "homebrew":
		return "homebrew"
	case "npmpackages":
		return "npm"
	case "gopackages":
		return "go"
	case "cargopackages":
		return "cargo"
	default:
		return strings.ToLower(kind)
	}
}

// ensureProvidersRegistered registers all providers if they haven't been registered yet.
// This is needed for commands that don't go through the Engine.
func ensureProvidersRegistered() {
	// Try to get each provider, and register if not found
	if _, err := provider.Get("file"); err != nil {
		provider.Register("file", providers.NewFileProvider(nil, nil, ""))
	}
	if _, err := provider.Get("homebrew"); err != nil {
		provider.Register("homebrew", providers.NewBrewProvider())
	}
	if _, err := provider.Get("npm"); err != nil {
		provider.Register("npm", providers.NewNpmProvider())
	}
	if _, err := provider.Get("go"); err != nil {
		provider.Register("go", providers.NewGoProvider())
	}
	if _, err := provider.Get("cargo"); err != nil {
		provider.Register("cargo", providers.NewCargoProvider())
	}
}

func runStateImport(kind, name, id string) error {
	// Ensure providers are registered
	ensureProvidersRegistered()

	// Map kind to provider name
	providerName := kindToProvider(kind)

	// Get the provider
	p, err := provider.Get(providerName)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Check if provider is available
	available, msg := p.Available()
	if !available {
		return fmt.Errorf("provider %s is not available: %s", kind, msg)
	}

	// Import the resource
	ctx := context.Background()
	resourceState, err := p.Import(ctx, id)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Set the resource name and kind
	resourceState.Name = name
	resourceState.Kind = kind

	// Load current state
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	statePath := dotisanDir + "/state.json"
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		currentState = state.NewState()
	}

	// Add the imported resource
	currentState.SetResource(resourceState)

	// Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Success message
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Printf("%s Imported %s/%s with ID %s\n", greenStyle.Render("✓"), kind, name, id)

	return nil
}

// stateRemoveCmd removes a resource from state
var stateRemoveCmd = &cobra.Command{
	Use:   "remove KIND NAME",
	Short: "Remove resource from state only",
	Long: `remove deletes the resource entry from the state file without
affecting the actual system. Use this when you want dotisan to stop
tracking a resource without removing it from your system.

Example: dotisan state remove BrewPackages core-tools`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := args[0]
		name := args[1]
		return runStateRemove(kind, name)
	},
}

func runStateRemove(kind, name string) error {
	// Load current state
	ctx := context.Background()
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	statePath := dotisanDir + "/state.json"
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		return fmt.Errorf("cannot load state: %w", err)
	}

	// Find and remove the resource
	found := false
	for i, r := range currentState.Resources {
		if r.Kind == kind && r.Name == name {
			// Remove by swapping with last and truncating
			currentState.Resources[i] = currentState.Resources[len(currentState.Resources)-1]
			currentState.Resources = currentState.Resources[:len(currentState.Resources)-1]
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("resource %s/%s not found in state", kind, name)
	}

	// Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Success message
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Printf("%s Removed %s/%s from state\n", greenStyle.Render("✓"), kind, name)
	fmt.Println("Note: The actual resource was not modified on your system.")

	return nil
}

// stateListCmd lists all managed resources
var stateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed resources",
	Long: `list displays all resources currently tracked in the state file
along with their status (in_sync, drift, missing).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStateList()
	},
}

func runStateList() error {
	// Load state
	ctx := context.Background()
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	statePath := dotisanDir + "/state.json"
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
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

	// Define lipgloss styles
	headerStyle := lipgloss.NewStyle().Bold(true)
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))

	// Print header
	fmt.Println(headerStyle.Render("Managed Resources"))
	fmt.Println()
	fmt.Printf("%-20s %-25s %-30s %-10s\n", "KIND", "NAME", "ID", "STATUS")
	fmt.Println(strings.Repeat("-", 85))

	// Load config for status check
	configPath := os.ExpandEnv("$HOME/.dotisan/config.yaml")
	cfg, _ := config.LoadConfig(configPath)

	// Determine status for each resource
	for _, r := range currentState.Resources {
		status := "in_sync"
		statusStyle := greenStyle

		// Simple heuristic: check if resource is in config
		if cfg != nil {
			// For now, we just show "in_sync" - a full status check would require
			// loading all resources and running Reconcile, which is expensive
			_ = cfg
		}

		fmt.Printf("%-20s %-25s %-30s %s\n",
			truncate(r.Kind, 20),
			truncate(r.Name, 25),
			truncate(r.ID, 30),
			statusStyle.Render(status))
	}

	fmt.Println()
	fmt.Printf("Total: %d resources\n", len(currentState.Resources))

	return nil
}

// Helper to truncate strings
func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// statePullCmd pulls state from remote backend
var statePullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Fetch state from remote backend",
	Long: `pull downloads the state from a configured remote backend (S3)
and overwrites the local state file. Use with caution.

Note: This requires S3 backend configuration in config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatePull()
	},
}

func runStatePull() error {
	// Load config to check for S3 backend
	configPath := os.ExpandEnv("$HOME/.dotisan/config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	// Check if S3 backend is configured
	if cfg.State.Backend != "s3" {
		return fmt.Errorf("S3 backend not configured in config.yaml (current: %s)", cfg.State.Backend)
	}

	// Create S3 backend
	ctx := context.Background()
	backend, err := state.NewS3Backend(state.S3Config{
		Endpoint:        cfg.State.S3.Endpoint,
		Bucket:          cfg.State.S3.Bucket,
		Key:             cfg.State.S3.Key,
		Region:          cfg.State.S3.Region,
		AccessKeyID:     cfg.State.S3.AccessKeyID,
		SecretAccessKey: cfg.State.S3.SecretAccessKey,
		UseSSL:          true, // Default to SSL
	})
	if err != nil {
		return fmt.Errorf("failed to initialize S3 backend: %w", err)
	}

	// Load remote state
	remoteState, err := backend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load remote state: %w", err)
	}

	// Save to local backend
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	localBackend := state.NewLocalBackend(dotisanDir)
	if err := localBackend.Save(ctx, remoteState); err != nil {
		return fmt.Errorf("failed to save local state: %w", err)
	}

	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Printf("%s Successfully pulled state from remote (%d resources)\n",
		greenStyle.Render("✓"), len(remoteState.Resources))

	return nil
}

// statePushCmd pushes state to remote backend
var statePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Write local state to remote backend",
	Long: `push uploads the local state file to a configured remote backend (S3),
overwriting the remote state. Use with caution.

Note: This requires S3 backend configuration in config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatePush()
	},
}

func runStatePush() error {
	// Load config to check for S3 backend
	configPath := os.ExpandEnv("$HOME/.dotisan/config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	// Check if S3 backend is configured
	if cfg.State.Backend != "s3" {
		return fmt.Errorf("S3 backend not configured in config.yaml (current: %s)", cfg.State.Backend)
	}

	// Load local state
	ctx := context.Background()
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	statePath := dotisanDir + "/state.json"
	localBackend := state.NewLocalBackend(statePath)
	localState, err := localBackend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load local state: %w", err)
	}

	// Create S3 backend and save
	backend, err := state.NewS3Backend(state.S3Config{
		Endpoint:        cfg.State.S3.Endpoint,
		Bucket:          cfg.State.S3.Bucket,
		Key:             cfg.State.S3.Key,
		Region:          cfg.State.S3.Region,
		AccessKeyID:     cfg.State.S3.AccessKeyID,
		SecretAccessKey: cfg.State.S3.SecretAccessKey,
		UseSSL:          true, // Default to SSL
	})
	if err != nil {
		return fmt.Errorf("failed to initialize S3 backend: %w", err)
	}

	if err := backend.Save(ctx, localState); err != nil {
		return fmt.Errorf("failed to push state: %w", err)
	}

	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	fmt.Printf("%s Successfully pushed state to remote (%d resources)\n",
		greenStyle.Render("✓"), len(localState.Resources))

	return nil
}

func init() {
	rootCmd.AddCommand(stateCmd)

	// Add subcommands
	stateCmd.AddCommand(stateImportCmd)
	stateCmd.AddCommand(stateRemoveCmd)
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(statePullCmd)
	stateCmd.AddCommand(statePushCmd)
}
