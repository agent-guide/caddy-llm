package rdb

import (
	"context"
	"fmt"
	"os"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RDBStore represents a store that uses a relational database.
type RDBStore struct {
	db     *gorm.DB
	logger *zap.Logger
}

type RDBConfig struct {
	DBType     string `json:"db_type"` // "sqlite"
	SQLitePath string `json:"sqlite_path,omitempty"`
}

// NewRDBStore creates a new RDBStore for the given dbtype.
func NewRDBStore(ctx context.Context, config *RDBConfig, logger *zap.Logger) (*RDBStore, error) {
	switch config.DBType {
	case "sqlite":
		return newSqliteConfigStore(ctx, config.SQLitePath, logger)
	default:
		return nil, fmt.Errorf("unsupported db type: %s", config.DBType)
	}
}

// newSqliteConfigStore creates a new SQLite config store.
func newSqliteConfigStore(ctx context.Context, sqlitePath string, logger *zap.Logger) (*RDBStore, error) {
	if _, err := os.Stat(sqlitePath); os.IsNotExist(err) {
		// Create DB file
		f, err := os.Create(sqlitePath)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
	}
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_busy_timeout=60000&_wal_autocheckpoint=1000&_foreign_keys=1", sqlitePath)
	logger.Debug("opening DB with dsn", zap.String("dsn", dsn))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: newGormLogger(*logger),
	})
	if err != nil {
		return nil, err
	}

	logger.Debug("sqlite db opened for RDBStore")
	return &RDBStore{db: db, logger: logger}, nil
}
