package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Standard Caddy modules
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	// LLM Gateway modules
	_ "github.com/agent-guide/caddy-llm/handler"
	_ "github.com/agent-guide/caddy-llm/llm"

	// LLM Providers (register via init())
	_ "github.com/agent-guide/caddy-llm/llm/provider/anthropic"
	_ "github.com/agent-guide/caddy-llm/llm/provider/gemini"
	_ "github.com/agent-guide/caddy-llm/llm/provider/ollama"
	_ "github.com/agent-guide/caddy-llm/llm/provider/openai"
	_ "github.com/agent-guide/caddy-llm/llm/provider/openrouter"
)

func main() {
	caddycmd.Main()
}
