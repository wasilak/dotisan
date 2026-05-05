package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wasilak/dotisan/pkg/style"

	"github.com/spf13/cobra"
)

var initForceFlag bool

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:          "init",
	SilenceUsage: true,
	Short:        "Initialize dotisan configuration",
	Long:         "Create default configuration and example files under ~/.config/dotisan/ (config.yaml, values.yaml, resources/)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit()
	},
}

func runInit() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "dotisan")
	resourcesDir := filepath.Join(configDir, "resources")
	configPath := filepath.Join(configDir, "config.yaml")
	valuesPath := filepath.Join(configDir, "values.yaml")

	welcomeBanner := style.SuccessBox.Render(
		style.Bold.Render("dotisan") + " - Your Dotfiles Manager\n\n" +
			style.DimStyle.Render("Version: 0.1.0") + " | " + style.DimStyle.Render("Manage your dotfiles with ease"),
	)
	fmt.Println(welcomeBanner)
	fmt.Println("Initializing...")

	// Check if already initialized
	if _, err := os.Stat(configDir); err == nil && !initForceFlag {
		fmt.Println(style.Iconf(style.IconWarning, style.Warning, "Directory %s already exists.", configDir))
		fmt.Println("Use --force to reinitialize (this will not overwrite existing files).")
		return nil
	}

	// Create directories
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", resourcesDir, err)
	}
	fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Created %s", configDir))
	fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Created %s", resourcesDir))

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
		fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Created %s", configPath))
	} else {
		fmt.Println(style.Iconf(style.IconWarning, style.Warning, "%s already exists (skipped)", configPath))
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
		fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Created %s", valuesPath))
	} else {
		fmt.Println(style.Iconf(style.IconWarning, style.Warning, "%s already exists (skipped)", valuesPath))
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
    # kind: HomeBrewPackages
    # metadata:
    #   name: cli-tools
    # spec:
    #   formulae:
    #     - name: ripgrep
    #     - name: fzf
#     - name: fd
`
		if err := os.WriteFile(sampleResourcePath, []byte(sampleContent), 0644); err != nil {
			return fmt.Errorf("failed to create sample.yaml: %w", err)
		}
		fmt.Println(style.Iconf(style.StyledIconSuccess, style.Success, "Created %s (example file)", sampleResourcePath))
	}

	fmt.Println()
	fmt.Println(style.Header.Render("Directory structure:"))
	fmt.Println("  " + style.DimStyle.Render("~/.config/dotisan/"))
	fmt.Println("  " + style.DimStyle.Render("├── config.yaml"))
	fmt.Println("  " + style.DimStyle.Render("├── values.yaml"))
	fmt.Println("  " + style.DimStyle.Render("└── resources/"))
	fmt.Println("      " + style.DimStyle.Render("└── sample.yaml"))
	fmt.Println()

	nextStepsBox := style.InfoBox.Render(
		style.Bold.Render("Next Steps") + "\n\n" +
			style.Success.Render("1. ") + "Edit values.yaml with your settings\n" +
			style.Success.Render("2. ") + "Create resources in resources/\n" +
			style.Success.Render("3. ") + "Run " + style.Info.Render("dotisan doctor") + " to verify\n" +
			style.Success.Render("4. ") + "Run " + style.Info.Render("dotisan plan") + " to preview",
	)
	fmt.Println(nextStepsBox)
	fmt.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initForceFlag, "force", false, "Force reinitialization (won't overwrite existing files)")
}
