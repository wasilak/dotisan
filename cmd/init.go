package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var initForceFlag bool

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:         "init",
	SilenceUsage: true,
	Short:       "Initialize dotisan configuration",
	Long: `init creates the default dotisan configuration directory and files:

  ~/.config/dotisan/              - Configuration directory
  ~/.config/dotisan/config.yaml   - Tool configuration
  ~/.config/dotisan/values.yaml   - Personal variables
  ~/.config/dotisan/resources/    - Resource definitions (YAML files)

This is a one-time setup command for new users.

Recommended structure:
  ~/.config/dotisan/
  ├── config.yaml
  ├── values.yaml
  └── resources/
      ├── shell.yaml
      ├── packages.yaml
      └── git.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func runInit() error {
	// Define lipgloss styles
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	boldStyle := lipgloss.NewStyle().Bold(true)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "dotisan")
	resourcesDir := filepath.Join(configDir, "resources")
	configPath := filepath.Join(configDir, "config.yaml")
	valuesPath := filepath.Join(configDir, "values.yaml")

	fmt.Println(boldStyle.Render("Initializing dotisan..."))
	fmt.Println()

	// Check if already initialized
	if _, err := os.Stat(configDir); err == nil && !initForceFlag {
		fmt.Printf("%s Directory %s already exists.\n", yellowStyle.Render("⚠"), configDir)
		fmt.Println("Use --force to reinitialize (this will not overwrite existing files).")
		return nil
	}

	// Create directories
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", resourcesDir, err)
	}
	fmt.Printf("%s Created %s\n", greenStyle.Render("✓"), configDir)
	fmt.Printf("%s Created %s\n", greenStyle.Render("✓"), resourcesDir)

	// Create default config.yaml if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) || initForceFlag {
		configContent := `# Dotisan Configuration
# This is the tool-level configuration file.
# For resource definitions, create YAML files in the resources/ subdirectory.

# Dotfiles/resources location (default: ~/.config/dotisan)
# dotfiles_root: ~/.config/dotisan

# State backend configuration
state:
  backend: local  # Options: local, s3
  path: ~/.config/dotisan/state.json
  
  # For S3 backend, uncomment and configure:
  # s3:
  #   endpoint: s3.amazonaws.com
  #   bucket: my-dotisan-state
  #   key: state.json
  #   region: us-east-1
  #   access_key_id: ${AWS_ACCESS_KEY_ID}
  #   secret_access_key: ${AWS_SECRET_ACCESS_KEY}
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to create config.yaml: %w", err)
		}
		fmt.Printf("%s Created %s\n", greenStyle.Render("✓"), configPath)
	} else {
		fmt.Printf("%s %s already exists (skipped)\n", yellowStyle.Render("⚠"), configPath)
	}

	// Create sample values.yaml if it doesn't exist
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) || initForceFlag {
		valuesContent := `# Dotisan Values
# Define variables here to use in your resource templates.
# Access them with {{ .Values.key }} in your YAML files.

# User information
# name: "Your Name"
# email: "your.email@example.com"
# github_username: "yourusername"

# Custom paths
# projects_dir: "{{ .Env.HOME }}/Projects"
# dotfiles_dir: "{{ .Env.HOME }}/.config/dotisan"

# Machine-specific settings
# editor: "nvim"
# shell: "zsh"
`
		if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
			return fmt.Errorf("failed to create values.yaml: %w", err)
		}
		fmt.Printf("%s Created %s\n", greenStyle.Render("✓"), valuesPath)
	} else {
		fmt.Printf("%s %s already exists (skipped)\n", yellowStyle.Render("⚠"), valuesPath)
	}

	// Create sample resource file
	sampleResourcePath := filepath.Join(resourcesDir, "sample.yaml")
	if _, err := os.Stat(sampleResourcePath); os.IsNotExist(err) || initForceFlag {
		sampleContent := `# Sample Resource Definition
# This is an example resource file. Copy and modify it, or create your own.
# Remove this file when you're ready to create real resources.

# Example 1: Inline source (good for small configs)
# ---
# apiVersion: github.com/wasilak/dotisan/v1
# kind: ManagedFile
# metadata:
#   name: zshrc
# spec:
#   source: |
#     # My zsh configuration
#     export EDITOR={{ .Values.editor | default "vim" }}
#     export EMAIL={{ .Values.email }}
#   destination: ~/.zshrc
#   mode: "0644"
#   template: true

# Example 2: External file source (better for IDE support)
# Create shell/zshrc.sh with your content, then:
# ---
# apiVersion: github.com/wasilak/dotisan/v1
# kind: ManagedFile
# metadata:
#   name: zshrc
# spec:
#   sourceFile: shell/zshrc.sh
#   destination: ~/.zshrc
#   mode: "0644"
#   template: true

# Example 3: Install packages with Homebrew
# ---
# apiVersion: github.com/wasilak/dotisan/v1
# kind: BrewPackages
# metadata:
#   name: cli-tools
# spec:
#   packages:
#     - name: ripgrep
#     - name: fzf
#     - name: fd
`
		if err := os.WriteFile(sampleResourcePath, []byte(sampleContent), 0644); err != nil {
			return fmt.Errorf("failed to create sample.yaml: %w", err)
		}
		fmt.Printf("%s Created %s (example file)\n", greenStyle.Render("✓"), sampleResourcePath)
	}

	fmt.Println()
	fmt.Println(boldStyle.Render("Directory structure:"))
	fmt.Println("  ~/.config/dotisan/")
	fmt.Println("  ├── config.yaml     # Tool configuration")
	fmt.Println("  ├── values.yaml     # Your personal variables")
	fmt.Println("  └── resources/      # Resource YAML files")
	fmt.Println("      └── sample.yaml # Example (remove when ready)")
	fmt.Println()
	fmt.Println(boldStyle.Render("Next steps:"))
	fmt.Println("  1. Edit ~/.config/dotisan/values.yaml with your personal settings")
	fmt.Println("  2. Create resource YAML files in ~/.config/dotisan/resources/")
	fmt.Println("  3. Remove sample.yaml when you're ready to add real resources")
	fmt.Println("  4. Run 'dotisan doctor' to verify your setup")
	fmt.Println("  5. Run 'dotisan plan' to see what would change")
	fmt.Println()
	fmt.Printf("%s Dotisan initialized successfully!\n", greenStyle.Render("✓"))

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initForceFlag, "force", false, "Force reinitialization (won't overwrite existing files)")
}
