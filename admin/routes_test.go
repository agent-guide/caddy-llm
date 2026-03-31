package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	configstoreintf "github.com/agent-guide/caddy-agent-gateway/configstore/intf"
	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"gorm.io/gorm"
)

type testConfigStore struct {
	routeStore       configstoreintf.RouteStorer
	localAPIKeyStore configstoreintf.LocalAPIKeyStorer
}

func (s *testConfigStore) GetCredentialStore(context.Context, configstoreintf.ConfigObjectDecoder) (configstoreintf.CredentialStorer, error) {
	return nil, nil
}

func (s *testConfigStore) GetProviderConfigStore() configstoreintf.ProviderConfigStorer {
	return nil
}

func (s *testConfigStore) GetLocalAPIKeyStore(context.Context, configstoreintf.ConfigObjectDecoder) (configstoreintf.LocalAPIKeyStorer, error) {
	return s.localAPIKeyStore, nil
}

func (s *testConfigStore) GetRouteStore(context.Context, configstoreintf.ConfigObjectDecoder) (configstoreintf.RouteStorer, error) {
	return s.routeStore, nil
}

type testRouteStore struct {
	items map[string]*routepkg.Route
}

func (s *testRouteStore) List(context.Context) ([]any, error) {
	out := make([]any, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

func (s *testRouteStore) Save(_ context.Context, id string, obj any) error {
	r, ok := obj.(*routepkg.Route)
	if !ok {
		return errors.New("unexpected type")
	}
	if s.items == nil {
		s.items = map[string]*routepkg.Route{}
	}
	cloned := *r
	s.items[id] = &cloned
	return nil
}

func (s *testRouteStore) Update(ctx context.Context, id string, obj any) error {
	if _, ok := s.items[id]; !ok {
		return gorm.ErrRecordNotFound
	}
	return s.Save(ctx, id, obj)
}

func (s *testRouteStore) Delete(_ context.Context, id string) error {
	delete(s.items, id)
	return nil
}

func (s *testRouteStore) Get(_ context.Context, id string) (any, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

type testLocalAPIKeyStore struct {
	items map[string]*routepkg.LocalAPIKey
}

func (s *testLocalAPIKeyStore) List(context.Context) ([]any, error) {
	out := make([]any, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

func (s *testLocalAPIKeyStore) Save(_ context.Context, key string, obj any) error {
	item, ok := obj.(*routepkg.LocalAPIKey)
	if !ok {
		return errors.New("unexpected type")
	}
	if s.items == nil {
		s.items = map[string]*routepkg.LocalAPIKey{}
	}
	cloned := *item
	s.items[key] = &cloned
	return nil
}

func (s *testLocalAPIKeyStore) Delete(_ context.Context, key string) error {
	delete(s.items, key)
	return nil
}

func (s *testLocalAPIKeyStore) Get(_ context.Context, key string) (any, error) {
	item, ok := s.items[key]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return item, nil
}

func TestRouteCRUD(t *testing.T) {
	handler := NewHandler(nil, &testConfigStore{
		routeStore: &testRouteStore{items: map[string]*routepkg.Route{}},
	}, nil)

	createBody, err := json.Marshal(routepkg.Route{
		ID:   "chat-prod",
		Name: "chat-prod",
		Targets: []routepkg.RouteTarget{{
			ProviderRef: "openai",
			Mode:        routepkg.TargetModeWeighted,
			Weight:      1,
		}},
	})
	if err != nil {
		t.Fatalf("marshal route: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/admin/routes", bytes.NewReader(createBody))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", createRec.Code, http.StatusCreated)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/routes/chat-prod", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", getRec.Code, http.StatusOK)
	}

	var got routepkg.Route
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode route: %v", err)
	}
	if got.ID != "chat-prod" {
		t.Fatalf("unexpected route id: got %q want %q", got.ID, "chat-prod")
	}
	if len(got.Targets) != 1 || got.Targets[0].ProviderRef != "openai" {
		t.Fatalf("unexpected targets: %#v", got.Targets)
	}
}

func TestLocalAPIKeyCRUD(t *testing.T) {
	handler := NewHandler(nil, &testConfigStore{
		localAPIKeyStore: &testLocalAPIKeyStore{items: map[string]*routepkg.LocalAPIKey{}},
	}, nil)

	body, err := json.Marshal(routepkg.LocalAPIKey{
		Key:             "lk-test",
		Name:            "test key",
		AllowedRouteIDs: []string{"chat-prod"},
	})
	if err != nil {
		t.Fatalf("marshal local api key: %v", err)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/admin/local_api_keys", bytes.NewReader(body))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", createRec.Code, http.StatusCreated)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/local_api_keys/lk-test", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", getRec.Code, http.StatusOK)
	}

	var got routepkg.LocalAPIKey
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode local api key: %v", err)
	}
	if got.Key != "lk-test" {
		t.Fatalf("unexpected local api key: got %q want %q", got.Key, "lk-test")
	}
	if len(got.AllowedRouteIDs) != 1 || got.AllowedRouteIDs[0] != "chat-prod" {
		t.Fatalf("unexpected allowed routes: %#v", got.AllowedRouteIDs)
	}
}
