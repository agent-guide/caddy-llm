package sqlite

import (
	"context"
	"testing"

	"github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRouteStoreReservedGroupColumn(t *testing.T) {
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	store, err := NewRouteStore(ctx, db, route.DecodeStoredRoute)
	if err != nil {
		t.Fatalf("new route store: %v", err)
	}

	want := &route.Route{
		ID:   "chat-prod",
		Name: "chat-prod",
		Targets: []route.RouteTarget{{
			ProviderRef: "openai",
			Mode:        route.TargetModeWeighted,
			Weight:      1,
		}},
	}

	if err := store.Save(ctx, want.ID, want); err != nil {
		t.Fatalf("save route: %v", err)
	}

	items, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list routes: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("list routes len = %d, want 1", len(items))
	}

	gotAny, err := store.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}

	got, ok := gotAny.(*route.Route)
	if !ok {
		t.Fatalf("get route type = %T, want *route.Route", gotAny)
	}
	if got.ID != want.ID {
		t.Fatalf("get route id = %q, want %q", got.ID, want.ID)
	}

	if err := store.Delete(ctx, want.ID); err != nil {
		t.Fatalf("delete route: %v", err)
	}

	items, err = store.List(ctx)
	if err != nil {
		t.Fatalf("list routes after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("list routes after delete len = %d, want 0", len(items))
	}
}
