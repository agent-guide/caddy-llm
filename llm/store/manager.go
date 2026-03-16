package store

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/agent-guide/caddy-llm/llm/store/intf"
	"github.com/agent-guide/caddy-llm/llm/store/rdb"
)

// Config holds the configuration for the store manager.
type Config struct {
	StoreType string         `json:"store_type"` // "rdb"
	RDBConfig *rdb.RDBConfig `json:"rdb_config,omitempty"`
}

// Manager holds all store instances keyed by name.
type Manager struct {
	stores map[string]interface{}
}

// NewManager creates a new Manager and initialises all stores from cfg.
func NewManager(ctx context.Context, cfg *Config, logger *zap.Logger) (*Manager, error) {
	m := &Manager{stores: make(map[string]interface{})}
	if err := m.initAllEntityStores(ctx, cfg, logger); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Manager) initAllEntityStores(ctx context.Context, cfg *Config, logger *zap.Logger) error {
	switch cfg.StoreType {
	case "rdb":
		rdbStore, err := rdb.NewRDBStore(ctx, cfg.RDBConfig, logger)
		if err != nil {
			return fmt.Errorf("init rdb store: %w", err)
		}
		credStore, err := rdb.NewCredentialStore(ctx, rdbStore)
		if err != nil {
			return fmt.Errorf("init credential store: %w", err)
		}
		m.stores["credential"] = credStore
	default:
		return fmt.Errorf("unsupported store type: %s", cfg.StoreType)
	}
	return nil
}

// CredentialStore returns the CredentialStorer, or an error if not initialised.
func (m *Manager) CredentialStore() (intf.CredentialStorer, error) {
	v, ok := m.stores["credential"]
	if !ok {
		return nil, fmt.Errorf("credential store not initialised")
	}
	s, ok := v.(intf.CredentialStorer)
	if !ok {
		return nil, fmt.Errorf("credential store has unexpected type %T", v)
	}
	return s, nil
}
