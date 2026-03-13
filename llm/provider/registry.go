package provider

import (
	"fmt"
	"sync"
)

// ProviderFactory creates a Provider instance from config.
type ProviderFactory func(config ProviderConfig) (Provider, error)

var (
	mu        sync.RWMutex
	factories = map[string]ProviderFactory{}
)

// RegisterProvider registers a provider factory by name.
func RegisterProvider(name string, factory ProviderFactory) {
	mu.Lock()
	defer mu.Unlock()
	factories[name] = factory
}

// NewProvider creates a provider by name using registered factories.
func NewProvider(config ProviderConfig) (Provider, error) {
	mu.RLock()
	factory, ok := factories[config.Name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", config.Name)
	}
	return factory(config)
}

// ListProviders returns the names of all registered providers.
func ListProviders() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	return names
}
