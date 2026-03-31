package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agent-guide/caddy-agent-gateway/llm/configstore/intf"
	"gorm.io/gorm"
)

type credentialRecord struct {
	sqliteJSONRecord
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (credentialRecord) TableName() string { return "credentials" }

// CredentialStore wraps RDBStore and implements intf.CredentialStorer.
type CredentialStore struct {
	*sqliteJSONStore
	userId string
}

// NewCredentialStore creates a CredentialStore and runs auto-migration.
func NewCredentialStore(ctx context.Context, db *gorm.DB, decodeCredential intf.ConfigObjectDecoder) (*CredentialStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&credentialRecord{}); err != nil {
		return nil, fmt.Errorf("credential store migrate: %w", err)
	}
	return &CredentialStore{
		sqliteJSONStore: newSQLiteJSONStore(db, credentialRecord{}.TableName(), "credential", decodeCredential),
		userId:          "default",
	}, nil
}

func (s *CredentialStore) ListByProviderName(ctx context.Context, providerName string) ([]any, error) {
	return s.sqliteJSONStore.ListByTagPrefix(ctx, s.qualifyProviderTag(providerName))
}

func (s *CredentialStore) Save(ctx context.Context, id string, tag string, obj any) (string, error) {
	return s.sqliteJSONStore.Save(ctx, id, s.qualifyProviderTag(tag), obj)
}

func (s *CredentialStore) Get(ctx context.Context, id string) (string, any, error) {
	tag, value, err := s.sqliteJSONStore.Get(ctx, id)
	if err != nil {
		return "", nil, err
	}
	return s.providerNameFromTag(tag), value, nil
}

func (s *CredentialStore) qualifyProviderTag(providerName string) string {
	providerName = strings.TrimSpace(providerName)
	if providerName == "" {
		return s.userId + "/"
	}
	return s.userId + "/" + providerName
}

func (s *CredentialStore) providerNameFromTag(tag string) string {
	prefix := s.userId + "/"
	if strings.HasPrefix(tag, prefix) {
		return strings.TrimPrefix(tag, prefix)
	}
	return tag
}
