package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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

func (s *VXApiKeyStore) List(ctx context.Context) ([]any, error) {
	var rows []vxAPIKeyRecord
	query := s.db.WithContext(ctx).Order("key asc")
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

func (s *VXApiKeyStore) Save(ctx context.Context, key string, obj any) error {
	record, err := newVXApiKeyRecord(key, obj)
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

func newVXApiKeyRecord(key string, obj any) (vxAPIKeyRecord, error) {
	if key == "" {
		return vxAPIKeyRecord{}, fmt.Errorf("vx api key is empty")
	}

	providerID, err := extractVXApiKeyProviderID(obj)
	if err != nil {
		return vxAPIKeyRecord{}, err
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

func extractVXApiKeyProviderID(obj any) (string, error) {
	if obj == nil {
		return "", fmt.Errorf("vx api key config is nil")
	}

	if providerID := extractProviderIDFromValue(reflect.ValueOf(obj)); providerID != "" {
		return providerID, nil
	}

	return "", fmt.Errorf("vx api key provider id is empty")
}

func extractProviderIDFromValue(v reflect.Value) string {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return ""
	}

	switch v.Kind() {
	case reflect.Map:
		for _, key := range []string{"provider_id", "ProviderId", "providerId", "providerID", "ProviderID"} {
			value := v.MapIndex(reflect.ValueOf(key))
			if !value.IsValid() {
				continue
			}
			for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
				if value.IsNil() {
					return ""
				}
				value = value.Elem()
			}
			if value.IsValid() && value.Kind() == reflect.String {
				return value.String()
			}
		}
	case reflect.Struct:
		for _, field := range []string{"ProviderId", "ProviderID"} {
			fv := v.FieldByName(field)
			if fv.IsValid() && fv.Kind() == reflect.String {
				return fv.String()
			}
		}
	}

	return ""
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
