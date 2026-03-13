package admin

import "net/http"

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
	}
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleListProviders(w http.ResponseWriter, r *http.Request)   { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleCreateProvider(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleGetProvider(w http.ResponseWriter, r *http.Request)     { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleUpdateProvider(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleDeleteProvider(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }

func (h *Handler) handleListMCPClients(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleAddMCPClient(w http.ResponseWriter, r *http.Request)    { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleGetMCPClient(w http.ResponseWriter, r *http.Request)    { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleUpdateMCPClient(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleRemoveMCPClient(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleListMCPTools(w http.ResponseWriter, r *http.Request)    { writeError(w, http.StatusNotImplemented, "not implemented") }

func (h *Handler) handleGetMemoryConfig(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleSetMemoryConfig(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleSearchMemory(w http.ResponseWriter, r *http.Request)    { writeError(w, http.StatusNotImplemented, "not implemented") }

func (h *Handler) handleListAgents(w http.ResponseWriter, r *http.Request)   { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleCreateAgent(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleGetAgent(w http.ResponseWriter, r *http.Request)     { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleUpdateAgent(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }
func (h *Handler) handleDeleteAgent(w http.ResponseWriter, r *http.Request)  { writeError(w, http.StatusNotImplemented, "not implemented") }

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotImplemented, "not implemented") }
