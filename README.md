# Prometheus MCP

![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/freepik-company/prometheus-mcp)
![GitHub](https://img.shields.io/github/license/freepik-company/prometheus-mcp)

A production-ready MCP (Model Context Protocol) server with integrated Prometheus client that enables seamless PromQL querying through the MCP protocol. Execute Prometheus queries, analyze metrics, and monitor your infrastructure directly from AI assistants like Claude, OpenAI, and more.

> **Built with [mcp-forge](https://github.com/achetronic/mcp-forge)** 🔧  
> This project is based on the excellent MCP server template by [@achetronic](https://github.com/achetronic), which provides production-ready OAuth authentication, JWT validation, and enterprise-grade features out of the box.

## Motivation
Monitoring and observability are crucial for modern applications, but accessing Prometheus metrics from AI assistants requires a seamless integration layer. This MCP server bridges that gap, allowing you to query Prometheus directly through natural language interactions with AI tools, making monitoring more accessible and intuitive.

## Features

- 🔍 **Complete Prometheus Integration**
  - Execute instant PromQL queries with `prometheus_query`
  - Perform range queries with `prometheus_range_query` 
  - List all available metrics with `prometheus_list_metrics`

- 🔐 **Enterprise Authentication Support**
  - HTTP Basic Authentication
  - Bearer Token Authentication (JWT/API tokens)
  - Multi-tenant support with `X-Scope-OrgId` header

- 🏢 **Multi-Platform Compatibility**
  - Vanilla Prometheus instances
  - Cortex multi-tenant setups
  - Thanos Query deployments
  - Grafana Cloud integration
  - Enterprise proxies with custom headers

- 🛡️ **Production Ready Security**
  - OAuth RFC 8414 and RFC 9728 compliant
  - JWT validation (delegated or local with JWKS)
  - Configurable access logs with field exclusion/redaction

- 🚀 **Infrastructure Features**
  - Docker containerization ready
  - Helm Chart for Kubernetes deployment
  - Structured JSON logging
  - Comprehensive error handling and troubleshooting

## Prometheus Configuration

### Basic Setup

Add Prometheus configuration to your `config.yaml`:

```yaml
prometheus:
  url: "http://localhost:9090"     # Prometheus server URL
  timeout: "30s"                   # Optional request timeout
```

### Advanced Configuration with Authentication

```yaml
prometheus:
  url: "https://prometheus.company.com"
  timeout: "30s"
  org_id: "tenant-1"               # X-Scope-OrgId header for multi-tenant systems
  auth:
    type: "basic"                  # Types: "basic", "token", or empty for no auth
    username: "prometheus-user"    # Username for basic auth
    password: "secret-password"    # Password for basic auth
```

### Bearer Token Configuration

```yaml
prometheus:
  url: "https://prometheus.company.com"
  timeout: "30s"
  org_id: "my-org-123"
  auth:
    type: "token"
    token: "eyJhbGciOiJIUzI1NiIs..."  # JWT or API token
```

### Configuration Options

- **`url`** (required): Prometheus server URL
- **`timeout`** (optional): Request timeout (e.g., "30s", "1m")
- **`org_id`** (optional): Value for `X-Scope-OrgId` header, useful for:
  - Cortex multi-tenant deployments
  - Thanos with tenant isolation
  - Prometheus behind multi-tenant proxy
- **`auth.type`** (optional): Authentication type
  - `"basic"`: HTTP Basic Authentication
  - `"token"`: Bearer Token Authentication
  - Empty or unspecified: No authentication
- **`auth.username`** and **`auth.password`**: Credentials for basic auth
- **`auth.token`**: Token for bearer authentication

### Common Use Cases

#### Prometheus Vanilla (No Authentication)
```yaml
prometheus:
  url: "http://localhost:9090"
```

#### Cortex Multi-Tenant
```yaml
prometheus:
  url: "http://cortex-gateway:8080/prometheus"
  org_id: "tenant-acme"
  auth:
    type: "basic"
    username: "cortex-user"
    password: "cortex-pass"
```

#### Thanos Query with Tenant Headers
```yaml
prometheus:
  url: "http://thanos-query:9090"
  org_id: "team-backend"
```

#### Grafana Cloud with API Token
```yaml
prometheus:
  url: "https://prometheus.grafana.net/api/prom"
  auth:
    type: "token" 
    token: "glc_eyJrIjoiN3..."  # Grafana Cloud API Key
```

## Available MCP Tools

### 1. `prometheus_query`

Execute instant PromQL queries against Prometheus.

**Parameters:**
- `query` (required): PromQL query to execute
- `time` (optional): Timestamp in RFC3339 format. Uses current time if not provided

**Example:**
```json
{
  "query": "up",
  "time": "2024-01-15T10:30:00Z"
}
```

### 2. `prometheus_range_query`

Execute PromQL range queries against Prometheus.

**Parameters:**
- `query` (required): PromQL query to execute
- `start` (required): Start time in RFC3339 format
- `end` (required): End time in RFC3339 format
- `step` (optional): Step duration (e.g., "30s", "1m", "5m"). Defaults to "1m"

**Example:**
```json
{
  "query": "rate(http_requests_total[5m])",
  "start": "2024-01-15T10:00:00Z",
  "end": "2024-01-15T11:00:00Z",
  "step": "1m"
}
```

### 3. `prometheus_list_metrics`

List all available metrics from Prometheus.

**Parameters:**
None.

**Example Response:**
```json
{
  "total_metrics": 245,
  "metrics": [
    "up",
    "prometheus_build_info",
    "http_requests_total"
  ]
}
```

## Deployment

### Production 🚀
Deploy to Kubernetes using the Helm chart located in the `chart/` directory.

---

Our recommendations for remote servers in production:

- Use a consistent hashring HTTP proxy in front of your MCP server when using MCP Sessions

- Use an HTTP proxy that performs JWT validation in front of the MCP instead of using the included middleware:
    - Protect your MCP exactly in the same way our middleware does, but with a super tested and scalable proxy instead
    - Improve your development experience as you don't have to do anything, just develop your MCP tools

- Use an OIDP that:
    - Cover Oauth Dynamic Client Registration
    - Is able to custom your JWT claims.

👉 [Keycloak](https://github.com/keycloak/keycloak) covers everything you need in the Oauth2 side

👉 [Istio](https://github.com/istio/istio) covers all you need to validate the JWT in front of your MCP

👉 [Hashrouter](https://github.com/achetronic/hashrouter) uses a configurable and truly consistent hashring to route the
traffic, so your sessions are safe with it. It has been tested under heavy load in production scenarios


## Getting Started

### Prerequisites
- Go 1.24+
- Access to a Prometheus server (local or remote)

### Quick Start

1. **Clone and build the project:**
   ```bash
   git clone <repository-url>
   cd prometheus-mcp
   make build
   ```

2. **Configure Prometheus connection:**
   
   Create a `config.yaml` file:
   ```yaml
   server:
     name: "prometheus-mcp"
     version: "1.0.0"
     transport:
       type: "stdio"  # or "http" for remote clients
   
   prometheus:
     url: "http://localhost:9090"
     timeout: "30s"
   ```

3. **Run the server:**
   ```bash
   # Stdio mode (for local clients like Claude Desktop)
   ./bin/prometheus-mcp -config config.yaml
   
   # HTTP mode (for remote clients)
   make run
   ```

### Development

To extend or modify the Prometheus MCP server:
- Prometheus tools are implemented in `internal/tools/tool_prometheus.go`
- Main client logic is in `internal/handlers/handlers.go`
- Configuration structures are in `api/config_types.go`

### Example Queries

**Basic monitoring:**
```promql
up                                    # Target status
prometheus_build_info                 # Prometheus version info
rate(http_requests_total[5m])         # HTTP request rate
```

**Resource monitoring:**
```promql
sum by (instance) (up)                        # Up targets by instance
increase(http_requests_total[1h])              # Request increase over 1h
topk(5, rate(http_requests_total[5m]))         # Top 5 highest request rates
```

**Kubernetes monitoring:**
```promql
kube_pod_status_phase{phase="Running"}         # Running pods
increase(kube_pod_container_status_restarts_total[30m]) > 5  # Pods with >5 restarts
```

### Configuration Examples

#### 🔗 Remote Clients (Claude Web, OpenAI)

Remote clients like Claude Web have different requirements than local ones. 
This project is fully ready for dealing with Claude Web with zero effort in your side.

In general, if you follow our recommendations on production, all the remote clients are covered 😊

> [!NOTE]  
> Hey! look at the configuration [here](./docs/config-http.yaml)

#### 💻 Local Clients (Claude Desktop, Cursor, VSCode)

Local clients configuration is commonly based in a JSON file with a specific standard structure. For example,
Claude Desktop can be configured by modifying the settings file called `claude_desktop_config.json` with the following sections:

##### Stdio Mode

If you want to use stdio as transport layer, it's recommended to compile your Go binary and then configure the client
as follows. This is recommended in local development as it is easy to work with.

Execute the following before configuring the client:

```console
make build
```

> [!IMPORTANT]
> When using Stdio transport, there is no protection between your client and the server, as they are both running locally

```json5
// file: claude_desktop_config.json

{
  "mcpServers": {
    "stdio": {
      "command": "/home/example/prometheus-mcp/bin/prometheus-mcp-linux-amd64",
      "args": [
        "--config",
        "/home/example/prometheus-mcp/docs/config-stdio.yaml"
      ]
    }
  }
}
```

##### HTTP Mode

It is possible to launch your MCP server using HTTP transport. As most of local clients doesn't support connecting to
remote servers natively, we use a package (`mcp-remote`) to act as an intermediate between the expected stdio, 
and the remote server, which is launched locally too.

This is ideal to work on all the features that will be deployed in production, as everything related to how remote clients 
will behave later is available, so everything can be truly tested

Execute the following before configuring the client:

```console
npm i mcp-remote && \
make run
```

```json5
// file: claude_desktop_config.json

{
 "mcpServers": {
   "local-proxy-remote": {
     "command": "npx",
     "args": [
       "mcp-remote",
       "http://localhost:8080/mcp",
       "--transport",
       "http-only",
       "--header",
       "Authorization: Bearer ${JWT}",
       "--header",
       "X-Validated-Jwt: ${JWT}"
     ],
     "env": {
       "JWT": "eyJhbGciOiJSUzI1NiIsImtpZCI6..."
     }
   }
 }
}
```


## Troubleshooting

### Common Issues

#### Error 401 Unauthorized
- Verify credentials in `auth.username` and `auth.password`
- For tokens, ensure `auth.token` is valid and hasn't expired
- Check if token has required permissions

#### Error 403 Forbidden  
- Verify `org_id` is correct for your tenant
- Confirm user has permissions for the specified tenant
- Check RBAC settings in multi-tenant setups

#### Connection Timeout Errors
- Increase `timeout` in configuration
- Verify network connectivity to Prometheus server
- Check firewall rules and network policies

#### Headers Not Recognized
- Some proxies filter custom headers
- Verify `X-Scope-OrgId` is supported by your installation
- Check proxy configuration for header forwarding

### Logging and Debugging

The server provides structured JSON logging with different levels:

**Successful initialization:**
```json
{
  "level": "INFO",
  "msg": "Prometheus client initialized successfully",
  "url": "https://prometheus.company.com",
  "auth_type": "basic",
  "org_id": "tenant-1"
}
```

**Authentication errors:**
```json
{
  "level": "ERROR", 
  "msg": "failed to execute Prometheus query",
  "error": "client error: 401 Unauthorized"
}
```

**Debug authentication (requires DEBUG log level):**
```json
{
  "level": "DEBUG",
  "msg": "Added basic auth to Prometheus request", 
  "username": "prometheus-user"
}
```

## 🌐 Documentation

For more information about MCP and related specifications:

👉 [MCP Authorization Requirements](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization#overview)

👉 [RFC 9728](https://datatracker.ietf.org/doc/rfc9728/)

👉 [MCP Go Documentation](https://mcp-go.dev/getting-started)

👉 [mcp-remote package](https://www.npmjs.com/package/mcp-remote)

👉 [Prometheus Query Documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/)


## 🤝 Contributing

All contributions are welcome! Whether you're reporting bugs, suggesting features, or submitting code — thank you! Here’s how to get involved:

▸ [Open an issue](https://github.com/freepik-company/prometheus-mcp/issues/new) to report bugs or request features

▸ [Submit a pull request](https://github.com/freepik-company/prometheus-mcp/pulls) to contribute improvements


## 📄 License

Prometheus MCP is licensed under the [Apache 2.0 License](./LICENSE).
