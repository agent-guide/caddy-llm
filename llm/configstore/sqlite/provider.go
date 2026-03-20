package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type providerConfigRecord struct {
	ID        string    `gorm:"primaryKey"`
	Tag       string    `gorm:"index;not null"`
	Config    string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (providerConfigRecord) TableName() string { return "providers" }

type ProviderConfigStore struct {
	db *gorm.DB
}

func NewProviderConfigStore(ctx context.Context, db *gorm.DB) (*ProviderConfigStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&providerConfigRecord{}); err != nil {
		return nil, fmt.Errorf("provider config store migrate: %w", err)
	}
	return &ProviderConfigStore{db: db}, nil
}

func (s *ProviderConfigStore) ListByTag(ctx context.Context, tag string) ([]any, error) {
	var rows []providerConfigRecord
	query := s.db.WithContext(ctx).Order("id asc")
	if tag != "" {
		query = query.Where("tag = ?", tag)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]any, 0, len(rows))
	for _, row := range rows {
		cfg, err := decodeProviderConfig(row)
		if err != nil {
			return nil, err
		}
		out = append(out, cfg)
	}
	return out, nil
}

func (s *ProviderConfigStore) Save(ctx context.Context, id string, tag string, obj any) (string, error) {
	record, err := newProviderConfigRecord(id, tag, obj)
	if err != nil {
		return "", err
	}

	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&record).Error; err != nil {
		return "", err
	}
	return record.ID, nil
}

func (s *ProviderConfigStore) Update(ctx context.Context, id string, obj any) error {
	var existing providerConfigRecord
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&existing).Error; err != nil {
		return err
	}

	data, err := marshalProviderConfig(obj)
	if err != nil {
		return err
	}

	return s.db.WithContext(ctx).
		Model(&providerConfigRecord{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"config":     string(data),
			"updated_at": time.Now(),
		}).Error
}

func (s *ProviderConfigStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).Where("id = ?", id).Delete(&providerConfigRecord{}).Error
}

func (s *ProviderConfigStore) Get(ctx context.Context, id string) (string, any, error) {
	var row providerConfigRecord
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		return "", nil, err
	}

	cfg, err := decodeProviderConfig(row)
	if err != nil {
		return "", nil, err
	}
	return row.Tag, cfg, nil
}

func newProviderConfigRecord(id string, tag string, obj any) (providerConfigRecord, error) {
	if id == "" {
		return providerConfigRecord{}, fmt.Errorf("provider config id is empty")
	}
	if tag == "" {
		return providerConfigRecord{}, fmt.Errorf("provider config tag is empty")
	}

	data, err := marshalProviderConfig(obj)
	if err != nil {
		return providerConfigRecord{}, err
	}

	return providerConfigRecord{
		ID:     id,
		Tag:    tag,
		Config: string(data),
	}, nil
}

func marshalProviderConfig(obj any) ([]byte, error) {
	if obj == nil {
		return nil, fmt.Errorf("provider config is nil")
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("provider config marshal: %w", err)
	}
	return data, nil
}

func decodeProviderConfig(row providerConfigRecord) (any, error) {
	var cfg any
	if err := json.Unmarshal([]byte(row.Config), &cfg); err != nil {
		return nil, fmt.Errorf("provider config %s unmarshal: %w", row.ID, err)
	}
	if obj, ok := cfg.(map[string]any); ok {
		if _, exists := obj["id"]; !exists {
			obj["id"] = row.ID
		}
		if _, exists := obj["tag"]; !exists {
			obj["tag"] = row.Tag
		}
		cfg = obj
	}
	return cfg, nil
}
