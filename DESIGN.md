# Caddy LLM AI Gateway - Detailed Design Document

## 1. Project Overview

### 1.1 Core Goals
Build an AI Gateway based on Caddy to provide AI Agent developers with:
- A unified API compatibility layer for multiple LLM providers
- Automatic configuration and management of MCP (Model Context Protocol)
- Seamless Memory integration without requiring changes to Agent code
- Comprehensive observability support
- Admin interface and Web UI

### 1.2 Design Principles
- **Modular**: Each component is independently extensible via Caddy's module system
- **Pluggable**: Clear core interfaces; third parties can add providers, memory backends, etc.
- **Zero-intrusion**: Agent developers can use MCP/Memory without modifying their code
- **High performance**: Leverages Caddy's high-performance HTTP processing

### 1.3 Module Hierarchy
```
caddy-llm/
├── llm  (Caddy App Module)
│   ├── provider  - LLM Provider management
│   ├── mcp       - MCP protocol support
│   ├── memory    - Memory management
│   ├── agent     - Agent mode orchestration
│   ├── config    - Configuration storage
│   └── auth      - CLI authentication (manager + authenticators)
│       ├── manager       - Credential lifecycle, refresh, selector
│       └── authenticator - Provider-specific login flows (Codex, Claude, …)
│
└── handler  (directory) → module ID: "http.handlers.llm"
    ├── llmapi    - LLM API compatibility layer
    ├── admin     - Admin API
    └── auth      - HTTP-level authentication & authorization (API key, RBAC)
```

---

## 2. Overall Architecture

### 2.1 System Architecture
```
┌─────────────────────────────────────────────────────────────────────┐
│                        Web UI (Next.js)                             │
│               (Deployed separately, calls Admin API)                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     http.handlers.llm (handler/)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │   llmapi     │  │    admin     │  │         auth             │  │
│  │ (API Router) │  │ (Management) │  │  (Authentication)        │  │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         llm (Caddy App)                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────────┐  │
│  │ provider │  │   mcp    │  │  memory  │  │       agent        │  │
│  │          │  │          │  │          │  │  (Orchestrator)    │  │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                         config                                │   │
│  │              (SQLite / PostgreSQL)                           │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       External Services                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────────┐  │
│  │ OpenAI   │  │Anthropic │  │  Gemini  │  │   MCP Servers      │  │
│  │          │  │          │  │          │  │  (stdio/SSE/WS)    │  │
│  └──────────┘  └──────────┘  └──────────┘  └────────────────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                          │
│  │ Vector DB│  │  Mem0    │  │  Others  │                          │
│  │ (Memory) │  │  API     │  │          │                          │
│  └──────────┘  └──────────┘  └──────────┘                          │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Data Flow

#### Standard LLM API call
```
Client Request → Auth → llmapi Handler → Provider → LLM Service → Response
```

#### Agent mode call
```
Client Request → Auth → llmapi Handler → Agent Orchestrator
                                              │
                    ┌─────────────────────────┼─────────────────────────┐
                    │                         │                         │
                    ▼                         ▼                         ▼
              MCP Integration           Memory Store            Provider Call
                    │                         │                         │
                    └─────────────────────────┼─────────────────────────┘
                                              │
                                              ▼
                                        Final Response
```

---

## 3. LLM App Module Detailed Design

### 3.1 Provider Submodule

#### 3.1.1 Interface Definition
```go
// Provider defines the interface for LLM providers.
type Provider interface {
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
    Stream(ctx context.Context, req *GenerateRequest) (*schema.StreamReader[*schema.Message], error)
    ListModels(ctx context.Context) ([]ModelInfo, error)
    Capabilities() ProviderCapabilities
}

// EmbeddingProvider is an optional interface for providers that support embeddings.
type EmbeddingProvider interface {
    Provider
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResponse, error)
}

// StatusError is implemented by errors that carry an HTTP status code.
// Allows the handler layer to make retry/degradation decisions:
// 401 → disable key, 429 → backoff, 503 → try next provider.
type StatusError interface {
    error
    StatusCode() int
}

// ProviderCapabilities describes what a provider instance supports.
type ProviderCapabilities struct {
    Streaming       bool
    Tools           bool
    Vision          bool
    Embeddings      bool
    ContextWindow   int
    MaxOutputTokens int
}

// ProviderConfig contains configuration for a provider instance.
type ProviderConfig struct {
    Name         string            `json:"name"`
    APIKey       string            `json:"api_key,omitempty"`
    BaseURL      string            `json:"base_url,omitempty"`
    DefaultModel string            `json:"default_model,omitempty"`
    Network      NetworkConfig     `json:"network"`
    Options      map[string]any    `json:"options,omitempty"`
}

// NetworkConfig controls HTTP client behavior for a provider.
type NetworkConfig struct {
    TimeoutSeconds    int               `json:"timeout_seconds,omitempty"`
    MaxRetries        int               `json:"max_retries,omitempty"`
    RetryDelaySeconds int               `json:"retry_delay_seconds,omitempty"`
    ProxyURL          string            `json:"proxy_url,omitempty"`
    ExtraHeaders      map[string]string `json:"extra_headers,omitempty"`
}

// GenerateRequest is the unified internal request format passed to providers.
type GenerateRequest struct {
    Model    string
    Messages []*schema.Message
    Options  []model.Option
}

// GenerateResponse is the unified internal response format returned by providers.
type GenerateResponse struct {
    Message *schema.Message
}
```

#### 3.1.2 Supported Providers
| Provider | Registry Key | Status | Notes |
|----------|-------------|--------|-------|
| OpenAI | `openai` | ✅ Implemented | `Generate/Stream` via Eino ChatModel; also implements `EmbeddingProvider` |
| Anthropic | `anthropic` | ✅ Implemented | `Generate/Stream` via Eino Claude; `ListModels` via native HTTP |
| Google Gemini | `gemini` | ✅ Implemented | `Generate/Stream` via Eino Gemini; `ListModels` via native HTTP |
| Ollama | `ollama` | ✅ Implemented | OpenAI-compatible base for `ListModels`/embed; chat via Eino |
| OpenRouter | `openrouter` | ✅ Implemented | OpenAI-compatible base for `ListModels`; chat via Eino |

#### 3.1.3 Shared HTTP Utilities
The provider layer now shares two main helper files/packages:

- **`llm/provider/httputil.go`**: `CheckResponse`, `BuildHTTPClient`, credential resolution, and per-request auth/header helpers.
- **`llm/provider/openaibase/`**: shared OpenAI-compatible base for `ListModels`, `Embed`, and common header handling. Used by OpenAI, Ollama, and OpenRouter.

Chat execution no longer goes through a shared hand-written HTTP chat base. `Generate/Stream` use Eino ChatModel implementations directly.

#### 3.1.4 Provider Registration
```go
// ProviderFactory creates a provider instance from config.
type ProviderFactory func(config ProviderConfig) (Provider, error)

// RegisterProvider registers a factory; called from provider init() functions.
func RegisterProvider(name string, factory ProviderFactory)

// NewProvider creates a provider instance by registered name.
func NewProvider(name string, config ProviderConfig) (Provider, error)

// ListProviders returns names of all registered providers.
func ListProviders() []string
```

---

### 3.2 MCP Submodule

#### 3.2.1 Protocol Support
Full MCP protocol support including:
- **stdio**: Local process communication
- **SSE (Server-Sent Events)**: HTTP long-polling
- **WebSocket**: Bidirectional communication

#### 3.2.2 Interface Definition
```go
// MCPManager manages MCP client connections
type MCPManager interface {
    AddClient(ctx context.Context, config MCPClientConfig) (*MCPClient, error)
    RemoveClient(ctx context.Context, clientID string) error
    GetClient(ctx context.Context, clientID string) (*MCPClient, error)
    ListClients(ctx context.Context) ([]*MCPClient, error)
    ListTools(ctx context.Context, clientID string) ([]Tool, error)
    ListResources(ctx context.Context, clientID string) ([]Resource, error)
    ReadResource(ctx context.Context, clientID, uri string) (*ResourceContent, error)
    ListPrompts(ctx context.Context, clientID string) ([]Prompt, error)
    GetPrompt(ctx context.Context, clientID, name string, args map[string]any) (*PromptResult, error)
}

// MCPClientConfig configuration for MCP client
type MCPClientConfig struct {
    ID         string
    Name       string
    Transport  TransportType     // stdio, sse, websocket
    Command    string            // for stdio
    URL        string            // for sse/websocket
    Env        map[string]string
    AutoAuth   bool
    AuthConfig *AuthConfig
}

// MCPClient represents an MCP server connection
type MCPClient interface {
    ID() string
    Name() string
    Status() ClientStatus
    CallTool(ctx context.Context, name string, args map[string]any) (*ToolResult, error)
    ListResources(ctx context.Context) ([]Resource, error)
    ReadResource(ctx context.Context, uri string) (*ResourceContent, error)
    ListPrompts(ctx context.Context) ([]Prompt, error)
    GetPrompt(ctx context.Context, name string, args map[string]any) (*PromptResult, error)
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
}
```

#### 3.2.3 Transport Layer
```go
// Transport defines the MCP transport interface
type Transport interface {
    Connect(ctx context.Context) error
    Close() error
    Send(ctx context.Context, msg *Message) error
    Receive() <-chan *Message
}

// StdioTransport implements stdio transport
type StdioTransport struct {
    cmd    *exec.Cmd
    stdin  io.WriteCloser
    stdout io.Reader
}

// SSETransport implements SSE transport
type SSETransport struct {
    url    string
    client *http.Client
    events <-chan *Event
}

// WebSocketTransport implements WebSocket transport
type WebSocketTransport struct {
    url  string
    conn *websocket.Conn
}
```

#### 3.2.4 Auto Authentication
```go
// AutoAuthProvider provides automatic authentication for MCP servers
type AutoAuthProvider interface {
    GetAuthHeaders(ctx context.Context, config *AuthConfig) (map[string]string, error)
    RefreshAuth(ctx context.Context, config *AuthConfig) error
}

// AuthConfig authentication configuration
type AuthConfig struct {
    Type      AuthType // api_key, oauth2, basic, custom
    APIKey    string
    OAuth2    *OAuth2Config
    BasicAuth *BasicAuthConfig
    Custom    map[string]any
}
```

---

### 3.3 Memory Submodule

#### 3.3.1 Core Interface
```go
// MemoryStore defines the interface for agent memory
type MemoryStore interface {
    Add(ctx context.Context, req *AddMemoryRequest) (*Memory, error)
    Get(ctx context.Context, id string) (*Memory, error)
    Update(ctx context.Context, id string, req *UpdateMemoryRequest) error
    Delete(ctx context.Context, id string) error
    Search(ctx context.Context, query string, opts *SearchOptions) ([]*Memory, error)
    SearchSimilar(ctx context.Context, embedding []float64, opts *SearchOptions) ([]*Memory, error)
    ListByAgent(ctx context.Context, agentID string, opts *ListOptions) ([]*Memory, error)
    ClearAgentMemory(ctx context.Context, agentID string) error
    BatchAdd(ctx context.Context, reqs []*AddMemoryRequest) ([]*Memory, error)
}

// Memory represents a memory entry
type Memory struct {
    ID        string
    AgentID   string
    Content   string
    Embedding []float64
    Metadata  map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
    Score     float64 // relevance score for search results
}

// SearchOptions search configuration
type SearchOptions struct {
    Limit     int
    Offset    int
    Threshold float64 // minimum similarity threshold
    Filters   map[string]any
}
```

#### 3.3.2 Storage Backends
```go
// MemoryBackend defines the storage backend interface
type MemoryBackend interface {
    MemoryStore
    Initialize(ctx context.Context, config *BackendConfig) error
    Close() error
    Health() error
}

// VectorStore interface for vector operations
type VectorStore interface {
    Insert(ctx context.Context, id string, embedding []float64, metadata map[string]any) error
    Search(ctx context.Context, embedding []float64, k int) ([]*SearchResult, error)
    Delete(ctx context.Context, id string) error
}
```

#### 3.3.3 Mem0 Adapter
```go
// Mem0Adapter adapts Mem0 API to MemoryStore interface
type Mem0Adapter struct {
    client    *mem0.Client
    projectID string
}

func (m *Mem0Adapter) Add(ctx context.Context, req *AddMemoryRequest) (*Memory, error) {
    mem0Req := &mem0.AddRequest{
        Content:  req.Content,
        UserID:   req.AgentID,
        Metadata: req.Metadata,
    }
    resp, err := m.client.Add(ctx, mem0Req)
    if err != nil {
        return nil, err
    }
    return convertToMemory(resp), nil
}
```

#### 3.3.4 Available Backends
| Backend | Module ID | Description |
|---------|-----------|-------------|
| SQLite + sqlite-vec | `llm.memory.sqlite` | Default, embedded |
| Mem0 API | `llm.memory.mem0` | Cloud service |
| PostgreSQL + pgvector | `llm.memory.postgres` | Production-grade |
| Chroma | `llm.memory.chroma` | Open-source vector DB |

---

### 3.4 Agent Submodule

#### 3.4.1 Orchestrator Interface
```go
// AgentOrchestrator orchestrates agent-mode requests
type AgentOrchestrator interface {
    Process(ctx context.Context, req *AgentRequest) (*AgentResponse, error)
}

// AgentRequest agent-mode request
type AgentRequest struct {
    SessionID    string
    AgentID      string
    Messages     []*schema.Message
    Tools        []*schema.ToolInfo
    Config       *AgentConfig
    EnableMCP    bool
    EnableMemory bool
    AutoToolCall bool
}

// AgentConfig agent configuration
type AgentConfig struct {
    MaxIterations int
    Timeout       time.Duration
    MemoryConfig  *MemoryConfig
    MCPConfig     *MCPConfig
}

// AgentResponse agent-mode response
type AgentResponse struct {
    SessionID string
    Messages  []*schema.Message
    Usage     Usage
}
```

#### 3.4.2 Automatic Tool Call Loop
```
1. Receive user request
2. Retrieve relevant memories (if Memory enabled)
3. Build context (system prompt + memories + user messages + MCP tools)
4. Call LLM
5. If tool_use returned:
   a. Execute tool call (MCP or custom tool)
   b. Add result to context
   c. Go back to step 4 (until max iterations or final reply)
6. Save new memories (if Memory enabled)
7. Return final reply
```

#### 3.4.3 Observability
```go
// ObservabilityData collected observability data
type ObservabilityData struct {
    TraceID       string
    Duration      time.Duration
    TokenUsage    TokenUsage
    ToolCalls     []ToolCallTrace
    MemoryAccess  []MemoryAccessTrace
    ProviderCalls []ProviderCallTrace
    Errors        []ErrorTrace
}

// TraceCollector collects traces
type TraceCollector interface {
    StartSpan(ctx context.Context, name string) (context.Context, Span)
    RecordEvent(ctx context.Context, event Event)
    Export(ctx context.Context) (*ObservabilityData, error)
}
```

---

### 3.5 Config Submodule

#### 3.5.1 Config Manager Interface
```go
// ConfigManager manages all configuration
type ConfigManager interface {
    GetProviderConfig(ctx context.Context, name string) (*ProviderConfig, error)
    SetProviderConfig(ctx context.Context, config *ProviderConfig) error
    ListProviders(ctx context.Context) ([]*ProviderConfig, error)
    GetMCPClientConfig(ctx context.Context, id string) (*MCPClientConfig, error)
    SetMCPClientConfig(ctx context.Context, config *MCPClientConfig) error
    ListMCPClientConfigs(ctx context.Context) ([]*MCPClientConfig, error)
    DeleteMCPClientConfig(ctx context.Context, id string) error
    GetMemoryConfig(ctx context.Context) (*MemoryBackendConfig, error)
    SetMemoryConfig(ctx context.Context, config *MemoryBackendConfig) error
    GetAgentConfig(ctx context.Context, id string) (*AgentConfig, error)
    SetAgentConfig(ctx context.Context, config *AgentConfig) error
    ListAgentConfigs(ctx context.Context) ([]*AgentConfig, error)
    GetGatewayConfig(ctx context.Context) (*GatewayConfig, error)
    SetGatewayConfig(ctx context.Context, config *GatewayConfig) error
}

// GatewayConfig gateway-level configuration
type GatewayConfig struct {
    ID              string
    Name            string
    DefaultProvider string
    AuthConfig      *AuthConfig
    RateLimit       *RateLimitConfig
    Logging         *LoggingConfig
}
```

#### 3.5.2 Config Store Interface
```go
// ConfigStore config storage interface
type ConfigStore interface {
    Get(ctx context.Context, key string, dest any) error
    Set(ctx context.Context, key string, value any) error
    Delete(ctx context.Context, key string) error
    List(ctx context.Context, prefix string) ([]string, error)
    Tx(ctx context.Context, fn func(tx ConfigStore) error) error
}
```

#### 3.5.3 Database Schema (SQLite)
```sql
CREATE TABLE providers (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    config     JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mcp_clients (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    transport  TEXT NOT NULL,
    config     JSON NOT NULL,
    status     TEXT DEFAULT 'inactive',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agents (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    config     JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    agent_id   TEXT REFERENCES agents(id),
    messages   JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE memories (
    id         TEXT PRIMARY KEY,
    agent_id   TEXT NOT NULL,
    content    TEXT NOT NULL,
    embedding  BLOB,
    metadata   JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE config (
    key        TEXT PRIMARY KEY,
    value      JSON NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE api_keys (
    id         TEXT PRIMARY KEY,
    key_hash   TEXT NOT NULL UNIQUE,
    name       TEXT NOT NULL,
    scopes     JSON NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP
);
```

---

### 3.6 Auth Submodule (`llm/auth/`)

The auth submodule provides CLI-level credential management — obtaining, storing, and refreshing API credentials from AI providers (e.g. simulating Codex or Claude desktop login flows). This is distinct from the HTTP-level auth in `handler/auth/`.

#### 3.6.1 Package Structure

| Package | Responsibility |
|---------|---------------|
| `llm/auth/manager` | Credential lifecycle: storage, selection, refresh scheduling, quota tracking |
| `llm/auth/authenticator` | Per-provider login flows (device code, PKCE browser-based OAuth, etc.) |

#### 3.6.2 Authenticator Interface
```go
// Authenticator is implemented by each provider-specific login flow.
type Authenticator interface {
    Provider() string
    Login(ctx context.Context) (*Credential, error)
    RefreshLead(ctx context.Context, cred *Credential) (*Credential, error)
}
```

#### 3.6.3 Credential Model
```go
// Credential represents a stored provider credential with lifecycle metadata.
type Credential struct {
    ID, Provider, Prefix, Label string
    Status      Status
    Disabled    bool
    Unavailable bool
    Attributes  map[string]any // provider-specific fields (api_key, base_url, priority, …)
    Metadata    map[string]any
    Quota       QuotaState
    ModelStates map[string]*ModelState
    // timestamps
    CreatedAt, UpdatedAt, LastRefreshedAt, NextRefreshAfter, NextRetryAfter time.Time
}
```

#### 3.6.4 Manager
```go
// Manager orchestrates credential lifecycle across all registered Authenticators.
type Manager struct {
    store     Store          // persistent credential storage
    selector  Selector       // chooses best credential for a request
    hook      Hook           // optional lifecycle hook (e.g. notify on refresh failure)
    // internal
    creds     map[string]*Credential
    refresher Refresher
}
```

#### 3.6.5 Implemented Authenticators
| Authenticator | File | Flow | Status |
|---------------|------|------|--------|
| `CodexAuthenticator` | `authenticator/codex.go` | OpenAI device-code OAuth | 🔧 Partial |
| `ClaudeAuthenticator` | `authenticator/claude.go` | Anthropic PKCE browser OAuth | 🔧 Partial |

---

## 4. HTTP Handler Module Detailed Design

### 4.1 LLM API Submodule

#### 4.1.1 Supported API Formats
| API Format | Route Prefix | Module |
|------------|--------------|--------|
| OpenAI | `/v1/chat/completions` | `http.handlers.llm.openai` |
| Anthropic | `/v1/messages` | `http.handlers.llm.anthropic` |
| Gemini | `/v1/models/{model}:generateContent` | `http.handlers.llm.gemini` |

#### 4.1.2 Request Conversion Flow
```
External API format → Format detection → Parser → Unified internal format → Provider → Response conversion
```

#### 4.1.3 OpenAI Handler Example
```go
// OpenAIHandler handles OpenAI-format requests
type OpenAIHandler struct {
    provider  Provider
    converter *OpenAIConverter
}

func (h *OpenAIHandler) HandleChatCompletions(w http.ResponseWriter, r *http.Request) error {
    var req openai.ChatCompletionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        return err
    }
    internalReq := h.converter.ToInternal(req)
    if req.Stream {
        return h.handleStream(w, r, internalReq)
    }
    resp, err := h.provider.Generate(r.Context(), internalReq)
    if err != nil {
        return err
    }
    return json.NewEncoder(w).Encode(h.converter.FromInternal(resp))
}
```

---

### 4.2 Admin Submodule

#### 4.2.1 Admin API Routes
| Resource | Method | Path | Description |
|----------|--------|------|-------------|
| **Providers** |
| | GET | `/admin/providers` | List all providers |
| | GET | `/admin/providers/{id}` | Get provider details |
| | POST | `/admin/providers` | Create provider |
| | PUT | `/admin/providers/{id}` | Update provider |
| | DELETE | `/admin/providers/{id}` | Delete provider |
| **MCP** |
| | GET | `/admin/mcp/clients` | List MCP clients |
| | GET | `/admin/mcp/clients/{id}` | Get client details |
| | POST | `/admin/mcp/clients` | Add MCP client |
| | PUT | `/admin/mcp/clients/{id}` | Update client |
| | DELETE | `/admin/mcp/clients/{id}` | Remove client |
| | GET | `/admin/mcp/clients/{id}/tools` | List client tools |
| **Memory** |
| | GET | `/admin/memory/config` | Get memory config |
| | PUT | `/admin/memory/config` | Set memory config |
| | GET | `/admin/memory/search` | Search memories |
| **Agents** |
| | GET | `/admin/agents` | List agents |
| | GET | `/admin/agents/{id}` | Get agent details |
| | POST | `/admin/agents` | Create agent |
| | PUT | `/admin/agents/{id}` | Update agent |
| | DELETE | `/admin/agents/{id}` | Delete agent |
| **Monitoring** |
| | GET | `/admin/health` | Health check |
| | GET | `/admin/metrics` | Prometheus metrics |
| | GET | `/admin/traces` | Trace data |

#### 4.2.2 Admin Handler
```go
// AdminHandler handles admin API requests
type AdminHandler struct {
    config    ConfigManager
    mcp       MCPManager
    memory    MemoryStore
    providers ProviderRegistry
}

func (h *AdminHandler) Routes() []Route {
    return []Route{
        {Method: "GET",    Path: "/providers",          Handler: h.ListProviders},
        {Method: "POST",   Path: "/providers",          Handler: h.CreateProvider},
        {Method: "GET",    Path: "/providers/{id}",     Handler: h.GetProvider},
        {Method: "PUT",    Path: "/providers/{id}",     Handler: h.UpdateProvider},
        {Method: "DELETE", Path: "/providers/{id}",     Handler: h.DeleteProvider},
        // ... other routes
    }
}
```

---

### 4.3 Auth Submodule (`handler/auth/`)

HTTP-level authentication and authorization. Validates incoming HTTP requests against stored API keys and enforces RBAC policies. This is distinct from `llm/auth/` which handles CLI credential acquisition.

#### 4.3.1 Authentication
```go
// Authenticator validates an HTTP request and returns the caller's identity.
type Authenticator interface {
    Authenticate(r *http.Request) (*Identity, error)
}

// IdentityType distinguishes callers by credential type.
type IdentityType string

const (
    IdentityTypeAPIKey IdentityType = "api_key"
    IdentityTypeUser   IdentityType = "user"
    IdentityTypeAgent  IdentityType = "agent"
)

// Identity represents an authenticated caller.
type Identity struct {
    ID       string
    Type     IdentityType
    Scopes   []string
    Metadata map[string]any
}

// extractAPIKey reads the key from the "x-api-key" header or "Authorization: Bearer …" header.
func extractAPIKey(r *http.Request) string { ... }
```

#### 4.3.2 Authorization (RBAC)
```go
// Authorizer enforces access control given a verified identity.
type Authorizer interface {
    Authorize(ctx context.Context, identity *Identity, resource, action string) error
}

// RBACAuthorizer role-based access control
type RBACAuthorizer struct {
    roles map[string]*Role
}

type Role struct {
    Name        string
    Permissions []Permission
}

type Permission struct {
    Resource string
    Actions  []string
}
```

---

## 5. Observability Design

### 5.1 Logging
```go
// StructuredLogger structured logging via zap
type StructuredLogger struct {
    logger *zap.Logger
}

func (l *StructuredLogger) Log(ctx context.Context, level LogLevel, msg string, fields ...Field) {
    if traceID := GetTraceID(ctx); traceID != "" {
        fields = append(fields, String("trace_id", traceID))
    }
    switch level {
    case LogLevelDebug:
        l.logger.Debug(msg, fields...)
    case LogLevelInfo:
        l.logger.Info(msg, fields...)
    case LogLevelWarn:
        l.logger.Warn(msg, fields...)
    case LogLevelError:
        l.logger.Error(msg, fields...)
    }
}
```

### 5.2 Metrics (Prometheus)
```go
type MetricsCollector struct {
    requestsTotal   *prometheus.CounterVec
    requestDuration *prometheus.HistogramVec
    tokensUsed      *prometheus.CounterVec
    toolCallsTotal  *prometheus.CounterVec
    errorsTotal     *prometheus.CounterVec
}

func (m *MetricsCollector) RecordRequest(provider, model, status string, duration time.Duration) {
    m.requestsTotal.WithLabelValues(provider, model, status).Inc()
    m.requestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

func (m *MetricsCollector) RecordTokens(provider, model string, input, output int) {
    m.tokensUsed.WithLabelValues(provider, model, "input").Add(float64(input))
    m.tokensUsed.WithLabelValues(provider, model, "output").Add(float64(output))
}
```

### 5.3 Tracing (OpenTelemetry)
```go
type Tracer struct {
    tracer trace.Tracer
}

func (t *Tracer) StartLLMSpan(ctx context.Context, provider, model string) (context.Context, Span) {
    return t.tracer.Start(ctx, fmt.Sprintf("llm.%s.%s", provider, model),
        trace.WithAttributes(
            attribute.String("provider", provider),
            attribute.String("model", model),
        ),
    )
}

func (t *Tracer) StartToolSpan(ctx context.Context, toolName string) (context.Context, Span) {
    return t.tracer.Start(ctx, fmt.Sprintf("tool.%s", toolName),
        trace.WithAttributes(
            attribute.String("tool.name", toolName),
        ),
    )
}
```

---

## 6. Project Directory Structure

Legend: ✅ implemented · 🔧 skeleton/stub · 📋 planned

```
caddy-llm/
├── cmd/
│   └── main.go                      # Entry point ✅
│
├── llm/                             # Caddy App Module (ID: "llm")
│   ├── app.go                       # App implementation ✅
│   │
│   ├── provider/                    # Provider submodule
│   │   ├── provider.go              # Interface + all shared types ✅
│   │   ├── registry.go              # Thread-safe provider factory registry ✅
│   │   ├── httputil.go              # Shared HTTP/client/credential helpers ✅
│   │   ├── eino.go                  # Eino option/message helpers ✅
│   │   ├── openaibase/              # Shared OpenAI-compatible base ✅
│   │   │   ├── types.go             #   Wire types for list-models/embeddings
│   │   │   └── provider.go          #   Base struct: ListModels/Embed/header handling
│   │   ├── openai/
│   │   │   └── provider.go          # Eino OpenAI chat + openaibase embed/list ✅
│   │   ├── anthropic/
│   │   │   ├── types.go             # Anthropic wire types ✅
│   │   │   └── provider.go          # Eino Claude chat + native list-models ✅
│   │   ├── gemini/
│   │   │   ├── types.go             # Gemini wire types ✅
│   │   │   └── provider.go          # Eino Gemini chat + native list-models ✅
│   │   ├── ollama/
│   │   │   └── provider.go          # Eino Ollama chat + openaibase list ✅
│   │   └── openrouter/
│   │       └── provider.go          # Eino OpenRouter chat + openaibase list ✅
│   │
│   ├── mcp/                         # MCP submodule
│   │   ├── manager.go               # MCPManager interface + Manager struct 🔧
│   │   ├── client.go                # Client interface + config/tool/resource types ✅
│   │   └── transport/
│   │       ├── transport.go         # Transport interface + JSON-RPC Message type ✅
│   │       ├── stdio.go             # stdio transport 🔧
│   │       ├── sse.go               # SSE transport 🔧
│   │       └── websocket.go         # WebSocket transport 🔧
│   │   # Planned: mcp/auth/, mcp/protocol/ 📋
│   │
│   ├── memory/                      # Memory submodule
│   │   ├── store.go                 # MemoryStore interface ✅
│   │   ├── backend.go               # Backend + VectorStore interfaces ✅
│   │   ├── embedding/
│   │   │   ├── embedder.go          # Embedder interface ✅
│   │   │   └── openai.go            # OpenAI embedder stub 🔧
│   │   ├── sqlite/
│   │   │   └── store.go             # SQLite + sqlite-vec stub 🔧
│   │   ├── mem0/
│   │   │   └── adapter.go           # Mem0 API adapter stub 🔧
│   │   # Planned: memory/postgres/, memory/chroma/ 📋
│   │
│   ├── agent/                       # Agent submodule
│   │   └── orchestrator.go          # Orchestrator skeleton 🔧
│   │   # Planned: executor.go, context.go, session.go 📋
│   │
│   ├── config/                      # Config submodule
│   │   ├── store.go                 # Store interface ✅
│   │   ├── manager.go               # Manager (GatewayConfig, ProviderConfig) ✅
│   │   └── sqlite/
│   │       └── store.go             # SQLite backend (modernc.org/sqlite) 🔧
│   │   # Planned: config/postgres/ 📋
│   │
│   └── auth/                        # Auth submodule (CLI credential management)
│       ├── manager/
│       │   ├── authenticator.go     # Authenticator interface ✅
│       │   ├── types.go             # Credential, QuotaState, ModelState, Error ✅
│       │   ├── manager.go           # Manager (credential lifecycle) 🔧
│       │   ├── store.go             # Store interface 🔧
│       │   ├── selector.go          # Selector interface 🔧
│       │   ├── status.go            # Status type 🔧
│       │   ├── persist_policy.go    # Persistence policy 🔧
│       │   ├── scheduler.go         # Background refresh scheduler 🔧
│       │   └── errors.go            # Error types 🔧
│       └── authenticator/
│           ├── codex.go             # CodexAuthenticator (OpenAI device-code flow) 🔧
│           └── claude.go            # ClaudeAuthenticator (Anthropic PKCE browser flow) 🔧
│
├── handler/                         # HTTP Handler Module (ID: "http.handlers.llm")
│   ├── module.go                    # Module registration ✅
│   ├── handler.go                   # Handler + Caddy provisioning 🔧
│   │
│   ├── llmapi/                      # LLM API compatibility layer
│   │   ├── router.go                # Routes /v1/messages → anthropic, /v1/chat → openai 🔧
│   │   ├── openai/
│   │   │   ├── handler.go           # OpenAI handler (501 stub) 🔧
│   │   │   └── converter.go         # OpenAI ↔ internal converter 🔧
│   │   ├── anthropic/
│   │   │   ├── handler.go           # Anthropic handler (501 stub) 🔧
│   │   │   └── converter.go         # Anthropic ↔ internal converter 🔧
│   │   # Planned: llmapi/gemini/ 📋
│   │
│   ├── admin/                       # Admin API submodule
│   │   ├── handler.go               # Admin handler (501 stub) 🔧
│   │   └── routes.go                # Route stubs 🔧
│   │   # Planned: provider_api.go, mcp_api.go, memory_api.go,
│   │   #          agent_api.go, monitor_api.go 📋
│   │
│   └── auth/                        # HTTP auth submodule (API key, RBAC)
│       ├── authenticator.go         # Authenticator interface + Identity + extractAPIKey ✅
│       ├── apikey.go                # API key auth stub 🔧
│       └── rbac.go                  # RBAC stub 🔧
│
├── internal/
│   ├── observability/
│   │   └── logger.go                # Structured zap logger 🔧
│   │   # Planned: metrics.go (Prometheus), tracer.go (OpenTelemetry) 📋
│   └── utils/
│       └── http.go                  # HTTP helpers 🔧
│       # Planned: json.go, crypto.go 📋
│
├── web/                             # Web UI (Next.js, separate deployment)
│   └── README.md                    # Placeholder 📋
│
├── examples/
│   └── test_client.go               # Basic test client
│   # Planned: examples/basic/, mcp/, agent/ 📋
│
├── go.mod
├── go.sum
├── Caddyfile
├── DESIGN.md                        # This document
└── README.md
```

---

## 7. Technology Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| HTTP framework | Caddy v2 | Core framework |
| Default database | SQLite + sqlite-vec | Embedded storage |
| Production database | PostgreSQL + pgvector | Optional |
| Embedding model | OpenAI text-embedding-3 | Vectorization |
| Logging | zap | Structured logging |
| Metrics | Prometheus | Monitoring |
| Tracing | OpenTelemetry | Distributed tracing |
| Frontend | Next.js | Web UI (separate deployment) |
| API docs | OpenAPI 3.0 | Interface documentation |

---

## 8. Implementation Roadmap

### Phase 1: Core Infrastructure ✅ Done
- [x] Project restructure (Caddy modules: `llm` app + `http.handlers.llm` handler)
- [x] Core interface definitions (`Provider`, `EmbeddingProvider`, `StatusError`)
- [x] Provider registry (thread-safe, `init()`-based registration)
- [x] Current providers implemented: OpenAI, Anthropic, Gemini, Ollama, OpenRouter
- [x] Shared `httputil.go` helpers (`CheckResponse`, HTTP client, credential/header handling)
- [x] Shared `openaibase` package (OpenAI-compatible list-models / embeddings / headers)
- [x] Config module: `Store` + `Manager` interfaces; SQLite backend skeleton
- [x] Memory module: `MemoryStore`, `Backend`, `VectorStore`, `Embedder` interfaces
- [x] MCP module: `Client`, `Transport` interfaces; JSON-RPC Message type
- [x] Auth module: `Authenticator` interface; `Credential` model with quota/model-state tracking; `CodexAuthenticator` + `ClaudeAuthenticator` skeletons
- [x] Handler module scaffolding (Caddy provisioning, Caddyfile parser, module registration)
- [x] HTTP auth: `Authenticator` interface, `Identity` type, `extractAPIKey` helper

### Phase 2: Core Features (in progress)
- [ ] LLM API handler — OpenAI format: request parsing → provider → SSE streaming response
- [ ] LLM API handler — Anthropic format: full wire compatibility
- [ ] LLM API handler — Gemini format: `/v1/models/{model}:generateContent`
- [ ] MCP transport implementations (stdio, SSE, WebSocket)
- [ ] MCP manager — connect/disconnect/tool-call lifecycle
- [ ] Memory backend — SQLite + sqlite-vec (vector insert, similarity search)
- [ ] Memory backend — OpenAI embedder implementation
- [ ] Admin API endpoints (CRUD for providers, MCP clients, agents)
- [ ] HTTP auth — API key validation against config store + RBAC enforcement
- [ ] Auth manager — credential store, selector, background refresh scheduler
- [ ] Auth authenticators — complete Codex device-code + Claude PKCE flows

### Phase 3: Advanced Features
- [ ] Agent orchestrator — complete tool call loop (max-iterations, MCP integration)
- [ ] Memory — Mem0 API adapter
- [ ] Observability — Prometheus metrics (`requests_total`, `tokens_used`, `tool_calls_total`)
- [ ] Observability — OpenTelemetry tracing
- [ ] Provider failover / retry (using `StatusError` codes: 429 → backoff, 503 → next provider)

### Phase 4: Production Backends
- [ ] PostgreSQL + pgvector memory backend
- [ ] PostgreSQL config store
- [ ] Chroma vector DB backend
- [ ] MCP auto-authentication (OAuth2, API key rotation)

### Phase 5: Web UI
- [ ] Next.js project setup (separate deployment, calls Admin API)
- [ ] Provider management UI
- [ ] MCP server management UI
- [ ] Agent configuration UI
- [ ] Monitoring dashboard (metrics + traces)

### Phase 6: Polish & Testing
- [ ] Unit tests for providers (mock HTTP server)
- [ ] Integration tests (real provider calls, configurable via env)
- [ ] API documentation (OpenAPI 3.0)
- [ ] Example configurations and agent code
