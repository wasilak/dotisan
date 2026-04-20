package provider

import (
	"fmt"
	"sync"
)

// Registry is the global provider registry.
// It maps provider names to Provider instances.
type Registry struct {
	providers map[string]Provider
	mu        sync.RWMutex
}

// globalRegistry is the singleton registry instance.
var globalRegistry = &Registry{
	providers: make(map[string]Provider),
}

// Register adds a provider to the global registry.
// Returns an error if a provider with the same name is already registered.
func Register(name string, p Provider) error {
	return globalRegistry.Register(name, p)
}

// Register adds a provider to the registry.
func (r *Registry) Register(name string, p Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[name]; exists {
		return fmt.Errorf("provider %q is already registered", name)
	}

	r.providers[name] = p
	return nil
}

// Get retrieves a provider by name from the global registry.
// Returns an error if the provider is not found.
func Get(name string) (Provider, error) {
	return globalRegistry.Get(name)
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %q not found", name)
	}

	return p, nil
}

// List returns all registered provider names from the global registry.
func List() []string {
	return globalRegistry.List()
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}

// GetAll returns all registered providers from the global registry.
func GetAll() []Provider {
	return globalRegistry.GetAll()
}

// GetAll returns all registered providers.
func (r *Registry) GetAll() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}

	return providers
}

// CheckAvailable checks all registered providers and returns availability info.
// Returns a map of provider name to (available, message) tuples.
func CheckAvailable() map[string]struct {
	Available bool
	Message   string
} {
	return globalRegistry.CheckAvailable()
}

// CheckAvailable checks all providers in the registry.
func (r *Registry) CheckAvailable() map[string]struct {
	Available bool
	Message   string
} {
	r.mu.RLock()
	providers := make(map[string]Provider, len(r.providers))
	for name, p := range r.providers {
		providers[name] = p
	}
	r.mu.RUnlock()

	results := make(map[string]struct {
		Available bool
		Message   string
	})

	for name, p := range providers {
		available, message := p.Available()
		results[name] = struct {
			Available bool
			Message   string
		}{
			Available: available,
			Message:   message,
		}
	}

	return results
}
