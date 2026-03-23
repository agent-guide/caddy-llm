# caddy-llm
A Caddy module for LLM API compatibility

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
