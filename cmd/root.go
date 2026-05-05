package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/wasilak/dotisan/pkg/config"

	"github.com/spf13/cobra"
	"github.com/wasilak/dotisan/cmd/middleware"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "dotisan",
	Short: "Declarative dotfiles management CLI",
	Long: `dotisan is a declarative dotfiles management CLI tool written in Go.

It treats a local developer environment like Terraform treats cloud infrastructure:
declare desired state in version-controlled config files, compute a diff against
current state, and apply changes — including removals.

Unlike chezmoi which applies changes forward but never cleans up, dotisan tracks
managed resources explicitly and handles removals as first-class operations.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to dotisan!")
		fmt.Println("Run 'dotisan --help' to see available commands.")
	},
}

// rootCtx is a process-global context that is cancelled on SIGINT/SIGTERM.
// It is initialized in Execute and used by subcommands to derive cancellable
// contexts for long-running operations.
var rootCtx context.Context

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Create a root context that is cancelled on SIGINT/SIGTERM and set it on
	// the root command so subcommands can derive from it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rootCtx = ctx
	rootCmd.SetContext(rootCtx)
	// No global app context anymore; callers should pass context explicitly.

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add persistent flag for log level which overrides config
	rootCmd.PersistentFlags().String("log-level", "", "Log level (debug, info, warn, error)")

	// Register plan command (restored)
	rootCmd.AddCommand(planCmd)

	// PersistentPreRun: initialize global slog logger using precedence: flag > config > default(info)
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Determine log level from flag
		lvlFlag, _ := cmd.Flags().GetString("log-level")

		// Load config to read configured log_level (if present)
		cfg, cfgErr := config.LoadConfigFromDefaultPath()
		if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
			slog.Warn("failed to load config", "err", cfgErr)
		}

		// Install spinner middleware so commands can opt-in to publish helpers
		// via the command context. This keeps wiring minimal and non-invasive.
		rootCmd.PersistentPreRunE = middleware.SpinnerMiddleware()

		chosen := "info"
		if strings.TrimSpace(lvlFlag) != "" {
			chosen = strings.ToLower(strings.TrimSpace(lvlFlag))
		} else if cfg != nil && strings.TrimSpace(cfg.LogLevel) != "" {
			chosen = strings.ToLower(strings.TrimSpace(cfg.LogLevel))
		}

		var level slog.Level
		switch chosen {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn", "warning":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}

		// Choose handler format according to application output setting.
		// If config requests JSON output, use a JSON handler for logs as well.
		var h slog.Handler
		if cfg != nil && strings.ToLower(strings.TrimSpace(cfg.UI.Output)) == "json" {
			h = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		} else {
			h = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
		}
		slog.SetDefault(slog.New(h))
	}
}
