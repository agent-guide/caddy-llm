package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agent-guide/caddy-llm/llm/authmanager/credential"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// credentialRecord is the GORM model for persisting Credential state.
type credentialRecord struct {
	ID        string    `gorm:"primaryKey"`
	Tag       string    `gorm:"index;not null"`
	Data      string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (credentialRecord) TableName() string { return "credentials" }

// CredentialStore wraps RDBStore and implements intf.CredentialStorer.
type CredentialStore struct {
	db *gorm.DB
}

// NewCredentialStore creates a CredentialStore and runs auto-migration.
func NewCredentialStore(ctx context.Context, db *gorm.DB) (*CredentialStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&credentialRecord{}); err != nil {
		return nil, fmt.Errorf("credential store migrate: %w", err)
	}
	return &CredentialStore{db: db}, nil
}

func (s *CredentialStore) ListByTag(ctx context.Context, tag string) ([]any, error) {
	var rows []credentialRecord
	query := s.db.WithContext(ctx).Order("id asc")
	if tag != "" {
		query = query.Where("tag = ?", tag)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	creds := make([]any, 0, len(rows))
	for _, row := range rows {
		c, err := decodeCredential(row)
		if err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, nil
}

func (s *CredentialStore) Save(ctx context.Context, id string, tag string, obj any) (string, error) {
	row, err := newCredentialRecord(id, tag, obj)
	if err != nil {
		return "", err
	}
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&row).Error; err != nil {
		return "", err
	}
	return row.ID, nil
}

// Delete removes the credential identified by id.
func (s *CredentialStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&credentialRecord{}).Error
}

func (s *CredentialStore) Get(ctx context.Context, id string) (string, any, error) {
	var row credentialRecord
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return "", nil, err
	}

	cred, err := decodeCredential(row)
	if err != nil {
		return "", nil, err
	}
	return row.Tag, cred, nil
}

func newCredentialRecord(id string, tag string, obj any) (credentialRecord, error) {
	if id == "" {
		return credentialRecord{}, fmt.Errorf("credential id is empty")
	}
	if tag == "" {
		return credentialRecord{}, fmt.Errorf("credential tag is empty")
	}
	if obj == nil {
		return credentialRecord{}, fmt.Errorf("credential config is nil")
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return credentialRecord{}, fmt.Errorf("credential marshal: %w", err)
	}

	return credentialRecord{
		ID:   id,
		Tag:  tag,
		Data: string(data),
	}, nil
}

func decodeCredential(row credentialRecord) (*credential.Credential, error) {
	var c credential.Credential
	if err := json.Unmarshal([]byte(row.Data), &c); err != nil {
		return nil, fmt.Errorf("credential %s unmarshal: %w", row.ID, err)
	}
	return &c, nil
}
