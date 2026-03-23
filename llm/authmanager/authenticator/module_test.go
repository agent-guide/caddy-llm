package authenticator

import (
	"testing"

	"github.com/agent-guide/caddy-llm/llm/authmanager/manager"
	"github.com/caddyserver/caddy/v2"
)

func TestBuiltinAuthenticatorsRegisterAsCaddyModules(t *testing.T) {
	for _, id := range []string{
		"llm.authenticators.codex",
		"llm.authenticators.claude",
		"llm.authenticators.gemini",
	} {
		info, err := caddy.GetModule(id)
		if err != nil {
			t.Fatalf("GetModule(%q) error = %v", id, err)
		}

		mod := info.New()
		if _, ok := mod.(manager.Authenticator); !ok {
			t.Fatalf("%q does not implement manager.Authenticator", id)
		}
	}
}
