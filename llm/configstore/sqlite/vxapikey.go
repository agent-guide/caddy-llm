package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
	"gorm.io/gorm"
)

type vxAPIKeyRecord struct {
	Key       string    `gorm:"primaryKey"`
	UserID    string    `gorm:"index;not null"`
	Config    string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (vxAPIKeyRecord) TableName() string { return "vx_api_keys" }

type VXApiKeyStore struct {
	*sqliteJSONStore
	userId string
}

func NewVXApiKeyStore(ctx context.Context, db *gorm.DB, decodeVXApiKey intf.ConfigObjectDecoder) (*VXApiKeyStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&vxAPIKeyRecord{}); err != nil {
		return nil, fmt.Errorf("vx api key store migrate: %w", err)
	}
	return &VXApiKeyStore{
		sqliteJSONStore: newSQLiteJSONStoreWithColumns(db, vxAPIKeyRecord{}.TableName(), "vx api key", "key", "user_id", "config", decodeVXApiKey),
		userId:          "default",
	}, nil
}

func (s *VXApiKeyStore) List(ctx context.Context) ([]any, error) {
	return s.sqliteJSONStore.ListByTagPrefix(ctx, s.userId)
}

func (s *VXApiKeyStore) Save(ctx context.Context, key string, obj any) error {
	_, err := s.sqliteJSONStore.Save(ctx, key, s.userId, obj)
	return err
}

func (s *VXApiKeyStore) Delete(ctx context.Context, key string) error {
	return s.sqliteJSONStore.Delete(ctx, key)
}

func (s *VXApiKeyStore) Get(ctx context.Context, key string) (any, error) {
	_, value, err := s.sqliteJSONStore.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	return value, nil
}
