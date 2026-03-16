package rdb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agent-guide/caddy-llm/llm/auth/credential"
	"gorm.io/gorm/clause"
)

// credentialRecord is the GORM model for persisting Credential state.
type credentialRecord struct {
	ID        string    `gorm:"primaryKey"`
	Data      string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (credentialRecord) TableName() string { return "credentials" }

// CredentialStore wraps RDBStore and implements intf.CredentialStorer.
type CredentialStore struct {
	rdb *RDBStore
}

// NewCredentialStore creates a CredentialStore and runs auto-migration.
func NewCredentialStore(ctx context.Context, rdb *RDBStore) (*CredentialStore, error) {
	if err := rdb.db.WithContext(ctx).AutoMigrate(&credentialRecord{}); err != nil {
		return nil, fmt.Errorf("credential store migrate: %w", err)
	}
	return &CredentialStore{rdb: rdb}, nil
}

// List returns all stored credentials.
func (s *CredentialStore) List(ctx context.Context) ([]*credential.Credential, error) {
	var rows []credentialRecord
	if err := s.rdb.db.WithContext(ctx).Find(&rows).Error; err != nil {
		return nil, err
	}
	creds := make([]*credential.Credential, 0, len(rows))
	for _, row := range rows {
		var c credential.Credential
		if err := json.Unmarshal([]byte(row.Data), &c); err != nil {
			return nil, fmt.Errorf("credential %s unmarshal: %w", row.ID, err)
		}
		creds = append(creds, &c)
	}
	return creds, nil
}

// Save persists the credential, replacing any existing record with the same ID.
// Returns the record ID as the storage key.
func (s *CredentialStore) Save(ctx context.Context, cred *credential.Credential) (string, error) {
	data, err := json.Marshal(cred)
	if err != nil {
		return "", fmt.Errorf("credential marshal: %w", err)
	}
	row := credentialRecord{
		ID:   cred.ID,
		Data: string(data),
	}
	if err := s.rdb.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&row).Error; err != nil {
		return "", err
	}
	return cred.ID, nil
}

// Delete removes the credential identified by id.
func (s *CredentialStore) Delete(ctx context.Context, id string) error {
	return s.rdb.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&credentialRecord{}).Error
}
