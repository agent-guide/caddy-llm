# caddy-llm

`caddy-llm` is a Caddy-native gateway for building and operating AI agents. It sits between agent runtimes and upstream model providers, exposing compatible LLM APIs while centralizing provider access, credential management, configuration storage, and operational control behind Caddy's module system.

The project is designed to make agent infrastructure easier to extend and easier to run in production. Instead of wiring provider clients, CLI logins, API compatibility layers, and admin endpoints directly into each agent application, developers can move those concerns into the gateway and reuse Caddy's configuration model, HTTP pipeline, and ecosystem.

The broader architecture also includes MCP, memory, and agent-oriented modules. Part of that surface is already implemented in the repository, while part is still being wired into the runtime and admin flows. This makes `caddy-llm` both a usable LLM gateway today and a foundation for a more complete AI agent gateway on top of the Caddy ecosystem.

## Features

- **Caddy-native gateway architecture**: built as a Caddy app plus HTTP handler modules, so gateway capabilities fit naturally into Caddy's configuration and runtime model.
- **Compatible LLM API exposure**: exposes familiar API shapes through `handle_llm_api`, reducing the amount of provider-specific code that agent applications need to carry.
- **Pluggable provider layer**: supports OpenAI, Anthropic, Gemini, Ollama, and OpenRouter through a common provider interface that is designed for extension.
- **Centralized credential lifecycle management**: manages provider credentials in one place, including persistence, selection, refresh, and operational state tracking.
- **Built-in CLI authentication flows**: includes authenticators for Codex/OpenAI, Claude/Anthropic, and Gemini, making CLI-derived credentials easier to operationalize in a gateway setting.
- **Persistent configuration storage**: uses SQLite as the default config store for credentials, provider configuration, and other gateway state.
- **Admin control surface**: provides an Admin API for health checks, provider configuration management, credential inspection, and CLI login operations.
- **Extensible runtime foundation**: includes interfaces and modules for MCP, memory, agent orchestration, authenticators, providers, and config stores, so the project can grow without collapsing into a monolith.

## Architecture

At a high level, the repository is organized around three layers:

- `llm/`: the core Caddy app module that owns shared gateway services such as provider registration, credential management, config storage, MCP, memory, and agent orchestration.
- `api/`: HTTP handlers that expose compatible LLM APIs to agent clients.
- `admin/`: HTTP handlers for operational and management endpoints under `/admin/*`.

This separation lets the gateway reuse Caddy's runtime and configuration model while keeping transport-facing handlers decoupled from internal provider, auth, and storage logic.

## Key Modules

### LLM Gateway App

The `llm` app module is the runtime backbone of the project. It provisions the config store, loads authenticators, restores persisted credentials, and exposes shared services to HTTP handlers registered elsewhere in the Caddy app graph.

### API Compatibility Layer

The `handle_llm_api` HTTP handler routes incoming requests to compatible API modules. The repository currently includes OpenAI-compatible and Anthropic-compatible handler implementations, allowing clients to keep familiar API shapes while the gateway controls upstream provider access.

### Provider System

Providers are registered behind a common interface, so model backends can be swapped or extended without changing the API layer. The current repository includes implementations for OpenAI, Anthropic, Gemini, Ollama, and OpenRouter, plus shared utilities for OpenAI-compatible behavior and HTTP client handling.

### Authentication and Credential Management

The gateway includes a credential manager with persistence, selection, status tracking, retry/backoff hooks, and refresh support. Built-in CLI authenticators currently cover Codex/OpenAI, Claude/Anthropic, and Gemini login flows, which allows the gateway to manage provider credentials centrally instead of scattering them across agent runtimes.

### Config Store

The default config store is backed by SQLite. It persists credentials, provider configuration, and related gateway state, giving the project a practical default for local development and self-hosted deployments while leaving room for additional backends later.

### MCP, Memory, and Agent Runtime

The repository already defines dedicated modules for MCP integration, memory storage, and agent orchestration. These areas are part of the intended core architecture, and the codebase already contains transport abstractions, memory interfaces, and orchestration scaffolding, but end-to-end runtime wiring is still incomplete in parts.

## Current Status

What is already usable:

- Running `caddy-llm` as a Caddy-based LLM gateway with a shared `llm` app module.
- Exposing compatible HTTP endpoints through `handle_llm_api`.
- Using the Admin API for health, provider config CRUD, credential listing, credential lookup, and CLI login flows.
- Registering and extending providers and authenticators through Caddy modules and Go interfaces.
- Persisting gateway state in SQLite.

What is still in progress:

- Full Admin API coverage for MCP, memory, metrics, and agent management.
- Complete runtime integration of MCP and memory into request execution paths.
- A production-ready web admin UI; the `web/` directory is currently a placeholder for a separate dashboard project.

## Why Caddy

Building on Caddy gives the project more than just an HTTP server. It provides a mature module system, flexible configuration loading, robust request handling, and a proven operational model. That makes `caddy-llm` a good fit for teams that want agent infrastructure to behave like gateway infrastructure: composable, inspectable, and maintainable.

## Roadmap Direction

The repository is moving toward a fuller AI agent gateway that can:

- unify multiple LLM providers behind stable APIs,
- manage CLI and API credentials centrally,
- integrate MCP tools and memory without pushing that complexity into each agent,
- expose operational controls and observability through admin surfaces,
- and remain extensible for third-party providers, stores, and runtime components.

Recommended Caddyfile usage:

Authenticators are configuration-driven now: if you do not declare an `authenticator` block, no CLI authenticator is enabled.

```caddy
{
    admin localhost:2019

    llm {
        config_store sqlite {
            path /var/lib/caddy/caddy-llm/configstore.db
        }

        authenticator codex {
            callback_port 1455
            no_browser false
        }

        authenticator claude {
            callback_port 54545
            no_browser false
        }
    }
}

localhost:8082 {
    route /v1/* {
        handle_llm_api {
            llm_api openai
            llm_api anthropic
        }
    }

    route /admin/* {
        handle_llm_admin
    }
}
```
