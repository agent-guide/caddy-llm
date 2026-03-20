package sqlite

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/agent-guide/caddy-llm/llm/configstore"
	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
)

type SQLiteConfigStore struct {
	logger          *zap.Logger
	sqlitePath      string
	db              *gorm.DB
	credentialStore *CredentialStore
	providerStore   *ProviderConfigStore
	vxAPIKeyStore   *VXApiKeyStore
}

type SQLiteConfigStoreConfig struct {
	SQLitePath string `json:"sqlite_path,omitempty"`
}

func init() {
	configstore.RegisterConfigStoreCreator("sqlite", NewSQLiteConfigStore)
}

// NewSqliteStore creates a new SQLite config store.
func NewSQLiteConfigStore(ctx context.Context, logger *zap.Logger, config any) (intf.ConfigStorer, error) {
	sqliteConfigStoreConfig, ok := config.(SQLiteConfigStoreConfig)
	if !ok {
		cfgPtr, ok := config.(*SQLiteConfigStoreConfig)
		if !ok || cfgPtr == nil {
			return nil, fmt.Errorf("invalid sqlite config store config type: %T", config)
		}
		sqliteConfigStoreConfig = *cfgPtr
	}

	dbPath := sqliteConfigStoreConfig.SQLitePath
	if dbPath == "" {
		return nil, fmt.Errorf("sqlite_path is required")
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, err
		}
		// Create DB file
		f, err := os.Create(dbPath)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
	}
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_busy_timeout=60000&_wal_autocheckpoint=1000&_foreign_keys=1", dbPath)
	logger.Debug("opening DB with dsn", zap.String("dsn", dsn))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: newGormLogger(*logger),
	})
	if err != nil {
		return nil, err
	}
	logger.Debug("sqlite db opened for SqliteStore")

	scs := &SQLiteConfigStore{sqlitePath: dbPath, db: db, logger: logger}

	credentialStore, err := NewCredentialStore(ctx, scs.db)
	if err != nil {
		return nil, fmt.Errorf("init credential store: %w", err)
	}
	scs.credentialStore = credentialStore

	providerStore, err := NewProviderConfigStore(ctx, scs.db)
	if err != nil {
		return nil, fmt.Errorf("init provider config store: %w", err)
	}
	scs.providerStore = providerStore

	vxAPIKeyStore, err := NewVXApiKeyStore(ctx, scs.db)
	if err != nil {
		return nil, fmt.Errorf("init vx api key store: %w", err)
	}
	scs.vxAPIKeyStore = vxAPIKeyStore

	return scs, nil
}

func (scs *SQLiteConfigStore) GetCredentialStore() intf.CredentialStorer {
	return scs.credentialStore
}

func (scs *SQLiteConfigStore) GetProviderConfigStore() intf.ProviderConfigStorer {
	return scs.providerStore
}

func (scs *SQLiteConfigStore) GetVXApiKeyStore() intf.VXApiKeyStorer {
	return scs.vxAPIKeyStore
}
