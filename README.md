# caddy-agent-gateway

`caddy-agent-gateway` is a Caddy-native AI gateway for agent and LLM workloads. It runs as a custom Caddy build, exposes compatible LLM HTTP APIs, centralizes provider access and credential management, and persists gateway configuration in SQLite by default.

The current codebase is already usable as a route-oriented LLM gateway. It also contains the early runtime scaffolding for MCP, memory, and agent orchestration, but those parts are not fully wired into request execution yet.

## What Exists Today

- A Caddy app module named `agent_gateway`
- HTTP handlers:
  - `handle_llm_api openai`
  - `handle_llm_api anthropic`
  - `agent_gateway_admin`
- Provider modules under `llm.providers.*`
- Authenticator modules under `llm.authenticators.*`
- SQLite-backed config storage for providers, routes, credentials, and local API keys
- Route-based target selection with weighted targets and route policy defaults
- Admin API CRUD for providers, routes, local API keys, and stored credentials
- Async CLI login flows for Codex/OpenAI, Claude, and Gemini authenticators

## Current Module Layout

- `gateway/`
  - Owns the `agent_gateway` Caddy app
  - Loads providers, authenticators, config store, and static routes
  - Builds runtime dependencies such as route loading, provider resolution, and local API key lookup
- `api/`
  - Registers `handle_llm_api`
  - Includes OpenAI-compatible and Anthropic-compatible ingress handlers
- `admin/`
  - Registers `agent_gateway_admin`
  - Exposes operational endpoints under `/admin/*`
- `llm/provider/`
  - Shared provider interfaces and provider implementations
  - Implemented providers: `openai`, `anthropic`, `gemini`, `ollama`, `openrouter`
- `llm/cliauth/`
  - Credential manager and provider-specific authenticators
  - Implemented authenticators: `codex`, `claude`, `gemini`
- `configstore/sqlite/`
  - Default persisted config backend
- `llm/mcp/`, `llm/memory/`, `llm/agent/`
  - Early interfaces and partial implementations for future runtime integration
- `web/`
  - Separate web UI work-in-progress, not yet the primary control plane

## Build

Build the custom Caddy binary directly:

```bash
go build -o caddy-agent-gateway ./cmd/main.go
```

Or use the existing Make target:

```bash
make build
```

The binary links the gateway app, admin handler, API handlers, and the built-in provider modules.

## Quick Start

Minimal `Caddyfile`:

```caddy
{
    admin localhost:2019

    agent_gateway {
        provider openai {
            api_key {$OPENAI_API_KEY}
            default_model gpt-4.1
        }

        config_store sqlite {
            path ./data/configstore.db
        }

        route openai-chat {
            require_local_api_key
            allowed_model gpt-4.1
            allowed_model gpt-4.1-mini
            target openai
        }
    }
}

:8082 {
    route /v1/* {
        handle_llm_api openai {
            route_id openai-chat
        }
    }

    route /admin/* {
        agent_gateway_admin
    }
}
```

Run it:

```bash
./caddy-agent-gateway run --config ./Caddyfile
```

## Caddyfile Model

The runtime is centered on the global `agent_gateway` block.

```caddy
{
    agent_gateway {
        provider <name> { ... }
        config_store sqlite { ... }
        authenticator <name> { ... }
        route <route-id> { ... }
    }
}
```

### Supported Global Subdirectives

- `provider <name> { ... }`
  - Loads a provider module from `llm.providers.<name>`
- `config_store sqlite { ... }`
  - Configures the default SQLite-backed config store
- `authenticator <name> { ... }`
  - Loads an authenticator from `llm.authenticators.<name>`
- `route <route-id> { ... }`
  - Declares a static gateway route

### Current Static Route Syntax

Static routes currently support these subdirectives:

- `route_name <name>`
- `require_local_api_key [true|false]`
- `allowed_model <model> [more-models...]`
- `target <provider> [weight]`

`target` entries are currently parsed as weighted targets. More advanced target conditions and policies exist in Go types and Admin API payloads, but the Caddyfile parser does not expose all of them yet.

## Request Flow

For a normal API call:

1. The HTTP handler selected by `handle_llm_api` receives the request.
2. The handler resolves `route_id`.
3. The gateway loads the route definition from the config store when available, otherwise from static app config.
4. If the route requires a local API key, the gateway validates the caller key.
5. The gateway resolves the target provider.
6. The compatible API handler converts the request into the internal provider request format.
7. The provider executes the upstream call and returns the translated response.

This means route and provider definitions managed through the Admin API can take effect without rebuilding the whole Caddy config.

## Providers

Built-in providers:

- `openai`
- `anthropic`
- `gemini`
- `ollama`
- `openrouter`

All providers implement the shared `provider.Provider` interface. Some providers also implement optional capabilities such as embeddings.

Custom providers can be added by shipping a Caddy module under `llm.providers.<name>` that implements the shared provider interfaces and is linked into the final Caddy build.

## Authentication and Credentials

There are two different credential layers in the project:

- Upstream provider credentials
  - Managed by the auth manager
  - Used when the gateway talks to OpenAI, Anthropic, Gemini, and other providers
- Local gateway API keys
  - Stored as `LocalAPIKey`
  - Used by agent clients to authenticate to the gateway itself

Built-in authenticators:

- `codex`
- `claude`
- `gemini`

If no `authenticator` block is declared, no CLI login flow is enabled.

## Admin API

The admin surface is mounted through `agent_gateway_admin` and currently includes:

- `GET /admin/health`
- Provider CRUD:
  - `GET /admin/providers`
  - `POST /admin/providers`
  - `GET /admin/providers/{id}`
  - `PUT /admin/providers/{id}`
  - `DELETE /admin/providers/{id}`
- Route CRUD:
  - `GET /admin/routes`
  - `POST /admin/routes`
  - `GET /admin/routes/{id}`
  - `PUT /admin/routes/{id}`
  - `DELETE /admin/routes/{id}`
- Local API key CRUD:
  - `GET /admin/local_api_keys`
  - `POST /admin/local_api_keys`
  - `GET /admin/local_api_keys/{key}`
  - `PUT /admin/local_api_keys/{key}`
  - `DELETE /admin/local_api_keys/{key}`
- Credential inspection:
  - `GET /admin/credentials`
  - `GET /admin/credentials/{id}`
  - `DELETE /admin/credentials/{id}`
- CLI login:
  - `POST /admin/cliauth/{cliname}`
  - `GET /admin/cliauth/{cliname}/status`

The route table also includes MCP, memory, agent, and metrics endpoints, but those handlers currently return `501 not implemented`.

## Admin API Examples

Create a provider record:

```bash
curl -X POST http://localhost:8082/admin/providers \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "openrouter",
    "tag": "openrouter",
    "config": {
      "base_url": "https://openrouter.ai/api/v1",
      "default_model": "openai/gpt-4o-mini"
    }
  }'
```

Create a route record:

```bash
curl -X POST http://localhost:8082/admin/routes \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "chat-prod",
    "name": "chat-prod",
    "targets": [
      { "provider_ref": "openrouter", "mode": "weighted", "weight": 1 }
    ],
    "policy": {
      "auth": { "require_local_api_key": true },
      "allowed_models": ["gpt-4o-mini"]
    }
  }'
```

Create a local API key:

```bash
curl -X POST http://localhost:8082/admin/local_api_keys \
  -H 'Content-Type: application/json' \
  -d '{
    "key": "lk-demo",
    "name": "demo key",
    "allowed_route_ids": ["chat-prod"]
  }'
```

Call the gateway:

```bash
curl http://localhost:8082/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'x-api-key: lk-demo' \
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "hello"}]
  }'
```

## Dynamic vs Static Configuration

There are two configuration sources:

- Static Caddyfile config under `agent_gateway`
- Persisted config in the SQLite config store

At runtime:

- Static providers are loaded during app provisioning.
- Persisted provider records can be resolved dynamically by ID.
- Static routes are registered at startup.
- If a handler specifies `route_id`, the gateway attempts to reload the latest persisted route definition for that ID on each request.

That design lets operators keep the Caddy app stable while changing route and provider records through the Admin API.

## Current Limits

These parts are real but incomplete:

- MCP transport packages and manager scaffolding exist, but Admin API integration is not implemented
- Memory interfaces and adapters exist, but request-path integration is partial
- Agent orchestration exists as an early loop around provider calls, but tool execution and memory integration are still TODOs
- The web dashboard exists as a separate Next.js app, but it is not yet the canonical operational surface

## Roadmap Direction

The repository is trending toward a fuller AI gateway that can:

- expose stable, provider-agnostic APIs for agent runtimes,
- centralize both gateway-side and upstream-side auth,
- integrate MCP and memory into gateway-managed execution,
- support richer routing policy and provider failover,
- and expose a complete operational control plane over Admin API and web UI.
