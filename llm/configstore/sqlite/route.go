package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
	"gorm.io/gorm"
)

type routeRecord struct {
	ID        string    `gorm:"primaryKey"`
	Group     string    `gorm:"index;not null"`
	Config    string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (routeRecord) TableName() string { return "routes" }

type RouteStore struct {
	*sqliteJSONStore
	group string
}

func NewRouteStore(ctx context.Context, db *gorm.DB, decodeRoute intf.ConfigObjectDecoder) (*RouteStore, error) {
	if err := db.WithContext(ctx).AutoMigrate(&routeRecord{}); err != nil {
		return nil, fmt.Errorf("route store migrate: %w", err)
	}

	return &RouteStore{
		sqliteJSONStore: newSQLiteJSONStoreWithColumns(db, routeRecord{}.TableName(), "route", "id", "group", "config", decodeRoute),
		group:           "default",
	}, nil
}

func (s *RouteStore) List(ctx context.Context) ([]any, error) {
	return s.sqliteJSONStore.ListByTag(ctx, s.group)
}

func (s *RouteStore) Save(ctx context.Context, id string, obj any) error {
	_, err := s.sqliteJSONStore.Save(ctx, id, s.group, obj)
	return err
}

func (s *RouteStore) Update(ctx context.Context, id string, obj any) error {
	_, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	return s.Save(ctx, id, obj)
}

func (s *RouteStore) Delete(ctx context.Context, id string) error {
	return s.sqliteJSONStore.Delete(ctx, id)
}

func (s *RouteStore) Get(ctx context.Context, id string) (any, error) {
	_, value, err := s.sqliteJSONStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return value, nil
}
