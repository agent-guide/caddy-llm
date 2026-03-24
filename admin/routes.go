package admin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/agent-guide/caddy-llm/llm/configstore/intf"
	"gorm.io/gorm"
)

// Route defines an admin API route.
type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
}

// Routes returns all admin API routes.
func (h *Handler) Routes() []Route {
	return []Route{
		// Health
		{Method: http.MethodGet, Path: "/admin/health", Handler: h.handleHealth},

		// Providers
		{Method: http.MethodGet, Path: "/admin/providers", Handler: h.handleListProviders},
		{Method: http.MethodPost, Path: "/admin/providers", Handler: h.handleCreateProvider},
		{Method: http.MethodGet, Path: "/admin/providers/{id}", Handler: h.handleGetProvider},
		{Method: http.MethodPut, Path: "/admin/providers/{id}", Handler: h.handleUpdateProvider},
		{Method: http.MethodDelete, Path: "/admin/providers/{id}", Handler: h.handleDeleteProvider},
		{Method: http.MethodGet, Path: "/admin/credentials", Handler: h.handleListCredentials},
		{Method: http.MethodGet, Path: "/admin/credentials/{id}", Handler: h.handleGetCredential},
		{Method: http.MethodDelete, Path: "/admin/credentials/{id}", Handler: h.handleDeleteCredential},

		// MCP
		{Method: http.MethodGet, Path: "/admin/mcp/clients", Handler: h.handleListMCPClients},
		{Method: http.MethodPost, Path: "/admin/mcp/clients", Handler: h.handleAddMCPClient},
		{Method: http.MethodGet, Path: "/admin/mcp/clients/{id}", Handler: h.handleGetMCPClient},
		{Method: http.MethodPut, Path: "/admin/mcp/clients/{id}", Handler: h.handleUpdateMCPClient},
		{Method: http.MethodDelete, Path: "/admin/mcp/clients/{id}", Handler: h.handleRemoveMCPClient},
		{Method: http.MethodGet, Path: "/admin/mcp/clients/{id}/tools", Handler: h.handleListMCPTools},

		// Memory
		{Method: http.MethodGet, Path: "/admin/memory/config", Handler: h.handleGetMemoryConfig},
		{Method: http.MethodPut, Path: "/admin/memory/config", Handler: h.handleSetMemoryConfig},
		{Method: http.MethodGet, Path: "/admin/memory/search", Handler: h.handleSearchMemory},

		// Agents
		{Method: http.MethodGet, Path: "/admin/agents", Handler: h.handleListAgents},
		{Method: http.MethodPost, Path: "/admin/agents", Handler: h.handleCreateAgent},
		{Method: http.MethodGet, Path: "/admin/agents/{id}", Handler: h.handleGetAgent},
		{Method: http.MethodPut, Path: "/admin/agents/{id}", Handler: h.handleUpdateAgent},
		{Method: http.MethodDelete, Path: "/admin/agents/{id}", Handler: h.handleDeleteAgent},

		// Metrics
		{Method: http.MethodGet, Path: "/admin/metrics", Handler: h.handleMetrics},

		// CLI Login
		{Method: http.MethodPost, Path: "/admin/clilogin/{cliname}", Handler: h.handleCLILogin},
		{Method: http.MethodGet, Path: "/admin/clilogin/{cliname}/status", Handler: h.handleCLILoginStatus},
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
