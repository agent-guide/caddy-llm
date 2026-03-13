package config

import (
	"context"
	"fmt"

	"github.com/agent-guide/caddy-llm/llm/provider"
)

// Manager provides typed access to all gateway configuration.
type Manager struct {
	store Store
}

// NewManager creates a new Config Manager.
func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

// GatewayConfig is gateway-level configuration.
type GatewayConfig struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	DefaultProvider string `json:"default_provider,omitempty"`
}

func (m *Manager) GetGatewayConfig(ctx context.Context) (*GatewayConfig, error) {
	var cfg GatewayConfig
	if err := m.store.Get(ctx, "gateway", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *Manager) SetGatewayConfig(ctx context.Context, cfg *GatewayConfig) error {
	return m.store.Set(ctx, "gateway", cfg)
}

func (m *Manager) GetProviderConfig(ctx context.Context, name string) (*provider.ProviderConfig, error) {
	var cfg provider.ProviderConfig
	if err := m.store.Get(ctx, "providers/"+name, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (m *Manager) SetProviderConfig(ctx context.Context, cfg *provider.ProviderConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("config: provider name is required")
	}
	return m.store.Set(ctx, "providers/"+cfg.Name, cfg)
}

func (m *Manager) ListProviders(ctx context.Context) ([]*provider.ProviderConfig, error) {
	keys, err := m.store.List(ctx, "providers/")
	if err != nil {
		return nil, err
	}
	configs := make([]*provider.ProviderConfig, 0, len(keys))
	for _, key := range keys {
		var cfg provider.ProviderConfig
		if err := m.store.Get(ctx, key, &cfg); err != nil {
			continue
		}
		configs = append(configs, &cfg)
	}
	return configs, nil
}

