# mcp-playground

A minimal MCP server used as a reference implementation for the `talk-libs/mcpserver` framework.

## Tools

| Tool  | Description                    |
|-------|--------------------------------|
| `sum` | Compute the sum of two integers |

## Environment Variables

| Variable    | Required | Default     | Description                                                                       |
|-------------|----------|-------------|-----------------------------------------------------------------------------------|
| `HTTP_HOST` | no       | `localhost` | Interface the HTTP transport binds to (`0.0.0.0` to accept external connections)  |
| `HTTP_PORT` | no       | `8080`      | Port the HTTP transport listens on                                                |

## Run

```bash
make dev
```

The HTTP transport listens on `$HTTP_HOST:$HTTP_PORT` (default `localhost:8080`).

## Authentication

This server supports **X-API-Key** and **OAuth 2.0** authentication.  
See [docs/mcp-server-authentication.md](../docs/mcp-server-authentication.md) for details.

## Security

This server includes built-in HTTP security hardening (rate limiting, path filtering, security headers, timeouts).  
See [docs/mcp-server-secured.md](../docs/mcp-server-secured.md) for configuration details.
