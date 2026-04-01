package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	routepkg "github.com/agent-guide/caddy-agent-gateway/gateway/route"
	"github.com/agent-guide/caddy-agent-gateway/configstore/intf"
	"gorm.io/gorm"
)

// Route defines an admin API route.
type Route struct {
	Method      string
	Path        string
	Handler     http.HandlerFunc
	RequireAuth bool
}

// Routes returns all admin API routes.
func (h *Handler) Routes() []Route {
	return []Route{
		// Health — public
		{Method: http.MethodGet, Path: "/admin/health", Handler: h.handleHealth},

		// Auth — login is public; logout and me require a valid session
		{Method: http.MethodPost, Path: "/admin/auth/login", Handler: h.handleLogin},
		{Method: http.MethodPost, Path: "/admin/auth/logout", Handler: h.handleLogout, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/auth/me", Handler: h.handleMe, RequireAuth: true},

		// Providers
		{Method: http.MethodGet, Path: "/admin/providers", Handler: h.handleListProviders, RequireAuth: true},
		{Method: http.MethodPost, Path: "/admin/providers", Handler: h.handleCreateProvider, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/providers/{id}", Handler: h.handleGetProvider, RequireAuth: true},
		{Method: http.MethodPut, Path: "/admin/providers/{id}", Handler: h.handleUpdateProvider, RequireAuth: true},
		{Method: http.MethodDelete, Path: "/admin/providers/{id}", Handler: h.handleDeleteProvider, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/routes", Handler: h.handleListRoutes, RequireAuth: true},
		{Method: http.MethodPost, Path: "/admin/routes", Handler: h.handleCreateRoute, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/routes/{id}", Handler: h.handleGetRoute, RequireAuth: true},
		{Method: http.MethodPut, Path: "/admin/routes/{id}", Handler: h.handleUpdateRoute, RequireAuth: true},
		{Method: http.MethodDelete, Path: "/admin/routes/{id}", Handler: h.handleDeleteRoute, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/local_api_keys", Handler: h.handleListLocalAPIKeys, RequireAuth: true},
		{Method: http.MethodPost, Path: "/admin/local_api_keys", Handler: h.handleCreateLocalAPIKey, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/local_api_keys/{key}", Handler: h.handleGetLocalAPIKey, RequireAuth: true},
		{Method: http.MethodPut, Path: "/admin/local_api_keys/{key}", Handler: h.handleUpdateLocalAPIKey, RequireAuth: true},
		{Method: http.MethodDelete, Path: "/admin/local_api_keys/{key}", Handler: h.handleDeleteLocalAPIKey, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/credentials", Handler: h.handleListCredentials, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/credentials/{id}", Handler: h.handleGetCredential, RequireAuth: true},
		{Method: http.MethodDelete, Path: "/admin/credentials/{id}", Handler: h.handleDeleteCredential, RequireAuth: true},

		// MCP
		{Method: http.MethodGet, Path: "/admin/mcp/clients", Handler: h.handleListMCPClients, RequireAuth: true},
		{Method: http.MethodPost, Path: "/admin/mcp/clients", Handler: h.handleAddMCPClient, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/mcp/clients/{id}", Handler: h.handleGetMCPClient, RequireAuth: true},
		{Method: http.MethodPut, Path: "/admin/mcp/clients/{id}", Handler: h.handleUpdateMCPClient, RequireAuth: true},
		{Method: http.MethodDelete, Path: "/admin/mcp/clients/{id}", Handler: h.handleRemoveMCPClient, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/mcp/clients/{id}/tools", Handler: h.handleListMCPTools, RequireAuth: true},

		// Memory
		{Method: http.MethodGet, Path: "/admin/memory/config", Handler: h.handleGetMemoryConfig, RequireAuth: true},
		{Method: http.MethodPut, Path: "/admin/memory/config", Handler: h.handleSetMemoryConfig, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/memory/search", Handler: h.handleSearchMemory, RequireAuth: true},

		// Agents
		{Method: http.MethodGet, Path: "/admin/agents", Handler: h.handleListAgents, RequireAuth: true},
		{Method: http.MethodPost, Path: "/admin/agents", Handler: h.handleCreateAgent, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/agents/{id}", Handler: h.handleGetAgent, RequireAuth: true},
		{Method: http.MethodPut, Path: "/admin/agents/{id}", Handler: h.handleUpdateAgent, RequireAuth: true},
		{Method: http.MethodDelete, Path: "/admin/agents/{id}", Handler: h.handleDeleteAgent, RequireAuth: true},

		// Metrics
		{Method: http.MethodGet, Path: "/admin/metrics", Handler: h.handleMetrics, RequireAuth: true},

		// CLI Login
		{Method: http.MethodPost, Path: "/admin/clilogin/{cliname}", Handler: h.handleCLILogin, RequireAuth: true},
		{Method: http.MethodGet, Path: "/admin/clilogin/{cliname}/status", Handler: h.handleCLILoginStatus, RequireAuth: true},
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type providerPayload struct {
	ID     string `json:"id"`
	Tag    string `json:"tag"`
	Config any    `json:"config"`
}

func (h *Handler) handleListProviders(w http.ResponseWriter, r *http.Request) {
	store := h.providerStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "provider store not configured")
		return
	}

	items, err := store.ListByName(r.Context(), r.URL.Query().Get("tag"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	store := h.providerStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "provider store not configured")
		return
	}

	var req providerPayload
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}
	if req.ID == "" || req.Tag == "" {
		writeError(w, http.StatusBadRequest, "id and tag are required")
		return
	}
	if req.Config == nil {
		writeError(w, http.StatusBadRequest, "config is required")
		return
	}

	id, err := store.Save(r.Context(), req.ID, req.Tag, req.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "tag": req.Tag, "config": req.Config})
}

func (h *Handler) handleGetProvider(w http.ResponseWriter, r *http.Request) {
	store := h.providerStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "provider store not configured")
		return
	}

	id := r.PathValue("id")
	tag, cfg, err := store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "tag": tag, "config": cfg})
}

func (h *Handler) handleUpdateProvider(w http.ResponseWriter, r *http.Request) {
	store := h.providerStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "provider store not configured")
		return
	}

	var req providerPayload
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}
	if req.Config == nil {
		writeError(w, http.StatusBadRequest, "config is required")
		return
	}

	id := r.PathValue("id")
	if err := store.Update(r.Context(), id, req.Config); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tag, cfg, err := store.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "tag": tag, "config": cfg})
}

func (h *Handler) handleDeleteProvider(w http.ResponseWriter, r *http.Request) {
	store := h.providerStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "provider store not configured")
		return
	}

	id := r.PathValue("id")
	if err := store.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (h *Handler) handleListRoutes(w http.ResponseWriter, r *http.Request) {
	store := h.routeStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "route store not configured")
		return
	}

	items, err := store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleCreateRoute(w http.ResponseWriter, r *http.Request) {
	store := h.routeStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "route store not configured")
		return
	}

	var route routepkg.Route
	if err := decodeJSON(r, &route); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}
	if route.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}
	if route.Name == "" {
		route.Name = route.ID
	}
	if len(route.Targets) == 0 {
		writeError(w, http.StatusBadRequest, "at least one target is required")
		return
	}
	route.Policy.Defaults()

	if err := store.Save(r.Context(), route.ID, &route); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, route)
}

func (h *Handler) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	store := h.routeStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "route store not configured")
		return
	}

	item, err := store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "route not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleUpdateRoute(w http.ResponseWriter, r *http.Request) {
	store := h.routeStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "route store not configured")
		return
	}

	var route routepkg.Route
	if err := decodeJSON(r, &route); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}
	id := r.PathValue("id")
	if route.ID == "" {
		route.ID = id
	}
	if route.ID != id {
		writeError(w, http.StatusBadRequest, "route id in body must match path")
		return
	}
	if route.Name == "" {
		route.Name = route.ID
	}
	if len(route.Targets) == 0 {
		writeError(w, http.StatusBadRequest, "at least one target is required")
		return
	}
	route.Policy.Defaults()

	if err := store.Update(r.Context(), id, &route); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "route not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	item, err := store.Get(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleDeleteRoute(w http.ResponseWriter, r *http.Request) {
	store := h.routeStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "route store not configured")
		return
	}

	id := r.PathValue("id")
	if err := store.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (h *Handler) handleListLocalAPIKeys(w http.ResponseWriter, r *http.Request) {
	store := h.localAPIKeyStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "local api key store not configured")
		return
	}

	items, err := store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleCreateLocalAPIKey(w http.ResponseWriter, r *http.Request) {
	store := h.localAPIKeyStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "local api key store not configured")
		return
	}

	var key routepkg.LocalAPIKey
	if err := decodeJSON(r, &key); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}
	if key.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if err := store.Save(r.Context(), key.Key, &key); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func (h *Handler) handleGetLocalAPIKey(w http.ResponseWriter, r *http.Request) {
	store := h.localAPIKeyStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "local api key store not configured")
		return
	}

	item, err := store.Get(r.Context(), r.PathValue("key"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "local api key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleUpdateLocalAPIKey(w http.ResponseWriter, r *http.Request) {
	store := h.localAPIKeyStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "local api key store not configured")
		return
	}

	var key routepkg.LocalAPIKey
	if err := decodeJSON(r, &key); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("decode request: %v", err))
		return
	}
	pathKey := r.PathValue("key")
	if key.Key == "" {
		key.Key = pathKey
	}
	if key.Key != pathKey {
		writeError(w, http.StatusBadRequest, "local api key in body must match path")
		return
	}

	if _, err := store.Get(r.Context(), pathKey); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "local api key not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := store.Save(r.Context(), key.Key, &key); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (h *Handler) handleDeleteLocalAPIKey(w http.ResponseWriter, r *http.Request) {
	store := h.localAPIKeyStore()
	if store == nil {
		writeError(w, http.StatusServiceUnavailable, "local api key store not configured")
		return
	}

	key := r.PathValue("key")
	if err := store.Delete(r.Context(), key); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "key": key})
}

func (h *Handler) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	if h.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth manager not configured")
		return
	}

	provider := r.URL.Query().Get("provider")
	items := h.authManager.List(provider)
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) handleGetCredential(w http.ResponseWriter, r *http.Request) {
	if h.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth manager not configured")
		return
	}

	id := r.PathValue("id")
	item := h.authManager.Get(id)
	if item == nil {
		writeError(w, http.StatusNotFound, "credential not found")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) handleListMCPClients(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleAddMCPClient(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleGetMCPClient(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleUpdateMCPClient(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleRemoveMCPClient(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleListMCPTools(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) handleGetMemoryConfig(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleSetMemoryConfig(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleSearchMemory(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) handleListAgents(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
func (h *Handler) handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (h *Handler) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	if h.authManager == nil {
		writeError(w, http.StatusServiceUnavailable, "auth manager not configured")
		return
	}

	id := r.PathValue("id")
	if err := h.authManager.Deregister(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (h *Handler) providerStore() intf.ProviderConfigStorer {
	if h.configStore == nil {
		return nil
	}
	return h.configStore.GetProviderConfigStore()
}

func (h *Handler) routeStore() intf.RouteStorer {
	if h.configStore == nil {
		return nil
	}
	store, err := h.configStore.GetRouteStore(context.Background(), routepkg.DecodeRoute)
	if err != nil {
		return nil
	}
	return store
}

func (h *Handler) localAPIKeyStore() intf.LocalAPIKeyStorer {
	if h.configStore == nil {
		return nil
	}
	store, err := h.configStore.GetLocalAPIKeyStore(context.Background(), routepkg.DecodeLocalAPIKey)
	if err != nil {
		return nil
	}
	return store
}
