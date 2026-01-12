package main

import (
	caddycmd "github.com/caddyserver/caddy/v2/cmd"

	// Plug in Caddy modules here
	_ "github.com/caddyserver/caddy/v2/modules/standard"

	// Our custom module - need to import the actual package with init()
	_ "github.com/agent-guide/caddy-llm-router"
)

func main() {
	caddycmd.Main()
}
