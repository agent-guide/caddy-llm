# caddy-llm
A Caddy module for LLM API compatibility

Recommended Caddyfile usage:

```caddy
{
    admin localhost:2019
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
