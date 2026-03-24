package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
)

type sqliteJSONRecord struct {
	ID        string    `gorm:"primaryKey"`
	Tag       string    `gorm:"index;not null"`
	Data      string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type sqliteJSONStore struct {
	db     *gorm.DB
	table  string
	kind   string
	decode intf.ConfigObjectDecoder
	// decode sqliteJSONDecoder
}

func newSQLiteJSONStore(db *gorm.DB, table string, kind string, decode intf.ConfigObjectDecoder) *sqliteJSONStore {
	return &sqliteJSONStore{
		db:     db,
		table:  table,
		kind:   kind,
		decode: decode,
	}
}

func (s *sqliteJSONStore) ListByTagPrefix(ctx context.Context, tagPrefix string) ([]any, error) {
	var rows []sqliteJSONRecord
	query := s.db.WithContext(ctx).Table(s.table).Order("id asc")
	if tagPrefix != "" {
		query = query.Where("tag LIKE ?", tagPrefix+"%")
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]any, 0, len(rows))
	for _, row := range rows {
		obj, err := s.decode([]byte(row.Data))
		if err != nil {
			return nil, err
		}
		out = append(out, obj)
	}
	return out, nil
}

func (s *sqliteJSONStore) ListByTag(ctx context.Context, tag string) ([]any, error) {
	var rows []sqliteJSONRecord
	query := s.db.WithContext(ctx).Table(s.table).Order("id asc")
	if tag != "" {
		query = query.Where("tag = ?", tag)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]any, 0, len(rows))
	for _, row := range rows {
		obj, err := s.decode([]byte(row.Data))
		if err != nil {
			return nil, err
		}
		out = append(out, obj)
	}
	return out, nil
}

func (s *sqliteJSONStore) Save(ctx context.Context, id string, tag string, obj any) (string, error) {
	row, err := s.newRecord(id, tag, obj)
	if err != nil {
		return "", err
	}
	if err := s.db.WithContext(ctx).
		Table(s.table).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&row).Error; err != nil {
		return "", err
	}
	return row.ID, nil
}

func (s *sqliteJSONStore) Delete(ctx context.Context, id string) error {
	return s.db.WithContext(ctx).
		Table(s.table).
		Where("id = ?", id).
		Delete(&sqliteJSONRecord{}).Error
}

func (s *sqliteJSONStore) Get(ctx context.Context, id string) (string, any, error) {
	var row sqliteJSONRecord
	if err := s.db.WithContext(ctx).Table(s.table).Where("id = ?", id).First(&row).Error; err != nil {
		return "", nil, err
	}

	obj, err := s.decode([]byte(row.Data))
	if err != nil {
		return "", nil, err
	}
	return row.Tag, obj, nil
}

func (s *sqliteJSONStore) newRecord(id string, tag string, obj any) (sqliteJSONRecord, error) {
	if id == "" {
		return sqliteJSONRecord{}, fmt.Errorf("%s id is empty", s.kind)
	}
	if tag == "" {
		return sqliteJSONRecord{}, fmt.Errorf("%s tag is empty", s.kind)
	}
	if obj == nil {
		return sqliteJSONRecord{}, fmt.Errorf("%s config is nil", s.kind)
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return sqliteJSONRecord{}, fmt.Errorf("%s marshal: %w", s.kind, err)
	}

	return sqliteJSONRecord{
		ID:   id,
		Tag:  tag,
		Data: string(data),
	}, nil
}
