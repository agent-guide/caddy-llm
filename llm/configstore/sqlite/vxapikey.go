package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type vxAPIKeyRecord struct {
	Key        string    `gorm:"primaryKey"`
	ProviderID string    `gorm:"index;not null"`
	Config     string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
}

func (vxAPIKeyRecord) TableName() string { return "vx_api_keys" }

type VXApiKeyStore struct {
	db *gorm.DB
}

func NewVXApiKeyStore(ctx context.Context, db *gorm.DB) (*VXApiKeyStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&vxAPIKeyRecord{}); err != nil {
		return nil, fmt.Errorf("vx api key store migrate: %w", err)
	}
	return &VXApiKeyStore{db: db}, nil
}

func (s *VXApiKeyStore) ListByProviderConfigID(ctx context.Context, providerID string) ([]any, error) {
	var rows []vxAPIKeyRecord
	query := s.db.WithContext(ctx).Order("key asc")
	if providerID != "" {
		query = query.Where("provider_id = ?", providerID)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]any, 0, len(rows))
	for _, row := range rows {
		cfg, err := decodeVXApiKey(row)
		if err != nil {
			return nil, err
		}
		out = append(out, cfg)
	}
	return out, nil
}

func (s *VXApiKeyStore) Save(ctx context.Context, key string, providerID string, obj any) error {
	record, err := newVXApiKeyRecord(key, providerID, obj)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&record).Error
}

func (s *VXApiKeyStore) Update(ctx context.Context, key string, obj any) error {
	var existing vxAPIKeyRecord
	if err := s.db.WithContext(ctx).Where("key = ?", key).First(&existing).Error; err != nil {
		return err
	}

	data, err := marshalVXApiKey(obj)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).
		Model(&vxAPIKeyRecord{}).
		Where("key = ?", key).
		Updates(map[string]any{
			"config":     string(data),
			"updated_at": time.Now(),
		}).Error
}

func (s *VXApiKeyStore) Delete(ctx context.Context, key string) error {
	return s.db.WithContext(ctx).Where("key = ?", key).Delete(&vxAPIKeyRecord{}).Error
}

func (s *VXApiKeyStore) Get(ctx context.Context, key string) (any, error) {
	var row vxAPIKeyRecord
	if err := s.db.WithContext(ctx).Where("key = ?", key).First(&row).Error; err != nil {
		return nil, err
	}
	return decodeVXApiKey(row)
}

func newVXApiKeyRecord(key string, providerID string, obj any) (vxAPIKeyRecord, error) {
	if key == "" {
		return vxAPIKeyRecord{}, fmt.Errorf("vx api key is empty")
	}
	if providerID == "" {
		return vxAPIKeyRecord{}, fmt.Errorf("vx api key provider id is empty")
	}

	data, err := marshalVXApiKey(obj)
	if err != nil {
		return vxAPIKeyRecord{}, err
	}

	return vxAPIKeyRecord{
		Key:        key,
		ProviderID: providerID,
		Config:     string(data),
	}, nil
}

func marshalVXApiKey(obj any) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("vx api key config is nil")
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("vx api key config marshal: %w", err)
	}
	return data, nil
}

func decodeVXApiKey(row vxAPIKeyRecord) (any, error) {
	var cfg any
	if err := json.Unmarshal([]byte(row.Config), &cfg); err != nil {
		return nil, fmt.Errorf("vx api key %s unmarshal: %w", row.Key, err)
	}
	if obj, ok := cfg.(map[string]any); ok {
		if _, exists := obj["key"]; !exists {
			obj["key"] = row.Key
		}
		if _, exists := obj["provider_id"]; !exists {
			obj["provider_id"] = row.ProviderID
		}
		cfg = obj
	}
	return cfg, nil
}
