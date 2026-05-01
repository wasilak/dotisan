// Package engine provides the core orchestration logic for dotisan.
package engine

import (
	"fmt"
	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/state"
)

// Engine orchestrates the plan and apply operations.
type Engine struct {
	Config          *config.Config
	TemplateContext *config.TemplateContext
	StateBackend    state.StateBackend
	Providers       map[string]provider.Provider
}

// NewEngine creates a new Engine with default configuration.
func NewEngine() (*Engine, error) {
	cfg, ctx, err := config.LoadComplete()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	backend, err := state.NewBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create state backend: %w", err)
	}

	providerMap := make(map[string]provider.Provider)

	// FileProvider
	fileProvider := providers.NewFileProvider(cfg.DotfilesRoot)
	providerMap[providerFile] = fileProvider
	provider.Register(providerFile, fileProvider)

	// BrewProvider
	brewProvider := providers.NewBrewProvider()
	providerMap[providerHomebrew] = brewProvider
	provider.Register(providerHomebrew, brewProvider)

	// NpmProvider
	npmProvider := providers.NewNpmProvider()
	providerMap[providerNpm] = npmProvider
	provider.Register(providerNpm, npmProvider)

	// GoProvider
	goProvider := providers.NewGoProvider()
	providerMap[providerGo] = goProvider
	provider.Register(providerGo, goProvider)

	// CargoProvider
	cargoProvider := providers.NewCargoProvider()
	providerMap[providerCargo] = cargoProvider
	provider.Register(providerCargo, cargoProvider)

	return &Engine{
		Config:          cfg,
		TemplateContext: ctx,
		StateBackend:    backend,
		Providers:       providerMap,
	}, nil
}
