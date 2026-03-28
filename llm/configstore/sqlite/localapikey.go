package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
	"gorm.io/gorm"
)

type localAPIKeyRecord struct {
	Key       string    `gorm:"primaryKey"`
	UserID    string    `gorm:"index;not null"`
	Config    string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (localAPIKeyRecord) TableName() string { return "local_api_keys" }

type LocalAPIKeyStore struct {
	*sqliteJSONStore
	userID string
}

func NewLocalAPIKeyStore(ctx context.Context, db *gorm.DB, decodeLocalAPIKey intf.ConfigObjectDecoder) (*LocalAPIKeyStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&localAPIKeyRecord{}); err != nil {
		return nil, fmt.Errorf("local api key store migrate: %w", err)
	}

	return &LocalAPIKeyStore{
		sqliteJSONStore: newSQLiteJSONStoreWithColumns(db, localAPIKeyRecord{}.TableName(), "local api key", "key", "user_id", "config", decodeLocalAPIKey),
		userID:          "default",
	}, nil
}

func (s *LocalAPIKeyStore) List(ctx context.Context) ([]any, error) {
	return s.sqliteJSONStore.ListByTagPrefix(ctx, s.userID)
}

func (s *LocalAPIKeyStore) Save(ctx context.Context, key string, obj any) error {
	_, err := s.sqliteJSONStore.Save(ctx, key, s.userID, obj)
	return err
}

func (s *LocalAPIKeyStore) Delete(ctx context.Context, key string) error {
	return s.sqliteJSONStore.Delete(ctx, key)
}

func (s *LocalAPIKeyStore) Get(ctx context.Context, key string) (any, error) {
	_, value, err := s.sqliteJSONStore.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return value, nil
}
