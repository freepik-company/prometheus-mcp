# Prometheus MCP

![GitHub go.mod Go version (subdirectory of monorepo)](https://img.shields.io/github/go-mod/go-version/freepik-company/prometheus-mcp)
![GitHub](https://img.shields.io/github/license/freepik-company/prometheus-mcp)

A production-ready MCP (Model Context Protocol) server with integrated Prometheus client that enables seamless PromQL querying through the MCP protocol. Execute Prometheus queries, analyze metrics, and monitor your infrastructure directly from AI assistants like Claude, OpenAI, and more.

> **Built with [mcp-forge](https://github.com/achetronic/mcp-forge)** 🔧  
> This project is based on the excellent MCP server template by [@achetronic](https://github.com/achetronic), which provides production-ready OAuth authentication, JWT validation, and enterprise-grade features out of the box.

## Motivation
Monitoring and observability are crucial for modern applications, but accessing Prometheus metrics from AI assistants requires a seamless integration layer. This MCP server bridges that gap, allowing you to query Prometheus directly through natural language interactions with AI tools, making monitoring more accessible and intuitive.

## Features

- 🔍 **Complete Metrics Integration**
  - Execute instant PromQL queries with `prometheus_query`
  - Perform range queries with `prometheus_range_query` 
  - List all available metrics with `prometheus_list_metrics`
  - Query any configured backend via the `backend` parameter

- 🔌 **Multi-Backend Support**
  - Configure multiple metrics backends (Prometheus, PMM, Thanos, VictoriaMetrics, etc.)
  - Switch between backends at query time with a single parameter
  - Add new backends with just YAML configuration — no code changes needed

- 🔐 **Enterprise Authentication Support**
  - HTTP Basic Authentication
  - Bearer Token Authentication (JWT/API tokens)
  - Multi-tenant support with `X-Scope-OrgId` header

- 🏢 **Multi-Platform Compatibility**
  - Vanilla Prometheus instances
  - Cortex multi-tenant setups
  - Thanos Query deployments
  - Grafana Cloud integration
  - PMM (Percona Monitoring and Management)
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

## Backends Configuration

All metrics backends are configured under the `backends` key as a map. Each entry's key becomes the backend name used in queries.

### Basic Setup

```yaml
backends:
  prometheus:
    url: "http://localhost:9090"
```

### Multiple Backends

```yaml
backends:
  prometheus:
    url: "http://localhost:9090"
    org_id: "tenant-1"
    available_orgs:
      - tenant-1
      - tenant-2

  pmm:
    url: "http://pmm:8080/prometheus"
    auth:
      type: "basic"
      username: "admin"
      password: "${PMM_PASSWORD}"

  thanos:
    url: "http://thanos-query:9090"
    auth:
      type: "token"
      token: "${THANOS_TOKEN}"
```

### Authentication

Each backend supports its own authentication independently:

#### Using Environment Variables (Recommended for Secrets)

```yaml
backends:
  prometheus:
    url: "https://prometheus.example.com"
    org_id: "tenant-1"
    auth:
      type: "basic"
      username: "prometheus-user"
      password: "${PROMETHEUS_PASSWORD}"
```

Both `${VAR}` and `$VAR` syntaxes are supported. Set the environment variable before starting the server:

```bash
export PROMETHEUS_PASSWORD="secret-password"
./prometheus-mcp --config config.yaml
```

#### Bearer Token Configuration

```yaml
backends:
  grafana-cloud:
    url: "https://prometheus.grafana.net/api/prom"
    auth:
      type: "token"
      token: "${GRAFANA_TOKEN}"
```

### Backend Configuration Options

- **`url`** (required): Metrics server URL
- **`org_id`** (optional): Value for `X-Scope-OrgId` header, useful for multi-tenant setups
- **`available_orgs`** (optional): List of available tenants (shown in tool descriptions)
- **`auth.type`** (optional): Authentication type
  - `"basic"`: HTTP Basic Authentication
  - `"token"`: Bearer Token Authentication
  - Empty or unspecified: No authentication
- **`auth.username`** and **`auth.password`**: Credentials for basic auth
- **`auth.token`**: Token for bearer authentication

### Common Use Cases

#### Prometheus Vanilla (No Authentication)
```yaml
backends:
  prometheus:
    url: "http://localhost:9090"
```

#### Cortex Multi-Tenant
```yaml
backends:
  cortex:
    url: "http://cortex-gateway:8080/prometheus"
    org_id: "tenant-acme"
    auth:
      type: "basic"
      username: "cortex-user"
      password: "cortex-pass"
```

#### Thanos Query with Tenant Headers
```yaml
backends:
  thanos:
    url: "http://thanos-query:9090"
    org_id: "team-backend"
```

#### Grafana Cloud with API Token
```yaml
backends:
  grafana-cloud:
    url: "https://prometheus.grafana.net/api/prom"
    auth:
      type: "token" 
      token: "glc_eyJrIjoiN3..."
```

#### PMM (Percona Monitoring and Management)
```yaml
backends:
  pmm:
    url: "http://pmm.example.com:8080/prometheus"
    auth:
      type: "basic"
      username: "admin"
      password: "${PMM_PASSWORD}"
```

## Multi-Tenant Support

Prometheus MCP supports **dynamic multi-tenant queries**, allowing you to query different tenants on-demand without restarting the server.

### Configuration

Set a **default tenant** and list of **available tenants** in your backend config:

```yaml
backends:
  prometheus:
    url: "http://mimir-gateway:8080/prometheus"
    org_id: "my-company"
    available_orgs:
      - my-company
      - my-company-slo
      - my-company-dev
```

When `available_orgs` is configured, AI agents will see these options in the tool description:

```
org_id: Optional tenant ID for multi-tenant Prometheus/Mimir. 
        Default: 'my-company'. 
        Available tenants: [my-company, my-company-slo, my-company-dev].
```

### Per-Query Tenant Override

Override the tenant for specific queries using the `org_id` parameter:

```json
{
  "backend": "prometheus",
  "query": "http_requests_total",
  "org_id": "my-company-slo"
}
```

### Behavior

- **Default tenant**: Uses `org_id` from config if no `org_id` is provided in the query
- **Explicit tenant**: Uses `org_id` from query parameter, overriding the config
- **No tenant**: If neither config nor query provides `org_id`, no `X-Scope-OrgId` header is sent
- **Non multi-tenant backends**: If `org_id` is provided for a backend without `org_id` or `available_orgs` in its config (e.g., PMM), the `X-Scope-OrgId` header is still sent but will likely be ignored by the server. A warning is logged in this case

## Available MCP Tools

All tools accept a `backend` parameter to specify which configured backend to query. If only one backend is configured, it is used by default.

### 1. `prometheus_query`

Execute instant PromQL queries against a metrics backend.

**Parameters:**
- `backend` (optional if single backend): Name of the backend to query
- `query` (required): PromQL query to execute
- `time` (optional): Timestamp in RFC3339 format. Uses current time if not provided
- `org_id` (optional): Tenant ID for multi-tenant setups. Overrides the default tenant from config

**Example:**
```json
{
  "backend": "prometheus",
  "query": "up",
  "time": "2024-01-15T10:30:00Z"
}
```

**Query PMM:**
```json
{
  "backend": "pmm",
  "query": "mysql_global_status_threads_connected"
}
```

### 2. `prometheus_range_query`

Execute PromQL range queries against a metrics backend.

**Parameters:**
- `backend` (optional if single backend): Name of the backend to query
- `query` (required): PromQL query to execute
- `start` (required): Start time in RFC3339 format
- `end` (required): End time in RFC3339 format
- `step` (optional): Step duration (e.g., "30s", "1m", "5m"). Defaults to "1m"
- `org_id` (optional): Tenant ID for multi-tenant setups. Overrides the default tenant from config

**Example:**
```json
{
  "backend": "prometheus",
  "query": "rate(http_requests_total[5m])",
  "start": "2024-01-15T10:00:00Z",
  "end": "2024-01-15T11:00:00Z",
  "step": "1m"
}
```

### 3. `prometheus_list_metrics`

List all available metrics from a metrics backend.

**Parameters:**
- `backend` (optional if single backend): Name of the backend to query
- `query` (optional): Glob pattern to filter metrics (e.g., 'redis*', '*cpu*')
- `org_id` (optional): Tenant ID for multi-tenant setups. Overrides the default tenant from config
- `limit` (optional): Maximum number of metrics to return. Defaults to 100
- `offset` (optional): Number of metrics to skip for pagination. Defaults to 0

**Example:**
```json
{
  "backend": "prometheus",
  "query": "http_*"
}
```

**Example Response:**
```json
{
  "total_metrics": 245,
  "returned": 100,
  "offset": 0,
  "limit": 100,
  "has_more": true,
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
- Access to a Prometheus-compatible metrics server (local or remote)

### Quick Start

1. **Clone and build the project:**
   ```bash
   git clone <repository-url>
   cd prometheus-mcp
   make build
   ```

2. **Configure backends:**
   
   Create a `config.yaml` file:
   ```yaml
   server:
     name: "prometheus-mcp"
     version: "1.0.0"
     transport:
       type: "stdio"  # or "http" for remote clients
   
   backends:
     prometheus:
       url: "http://localhost:9090"
   ```

3. **Run the server:**
   ```bash
   ./bin/prometheus-mcp -config config.yaml
   ```

### Development

To extend or modify the Prometheus MCP server:
- Tool handlers are in `internal/tools/tool_prometheus_*.go`
- Backend client logic is in `internal/handlers/handlers.go`
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

#### Unknown Backend Error
- Verify the `backend` parameter matches a key in your `backends` configuration
- Check that the backend URL is accessible
- Verify the backend is properly initialized in the server logs

### Logging and Debugging

The server provides structured JSON logging with different levels:

**Successful initialization:**
```json
{
  "level": "INFO",
  "msg": "Backend client initialized",
  "backend": "prometheus",
  "url": "https://prometheus.company.com",
  "auth_type": "basic",
  "org_id": "tenant-1"
}
```

**Authentication errors:**
```json
{
  "level": "ERROR", 
  "msg": "failed to execute query on backend \"prometheus\"",
  "error": "client error: 401 Unauthorized"
}
```

**Debug authentication (requires DEBUG log level):**
```json
{
  "level": "DEBUG",
  "msg": "Added basic auth to request", 
  "backend": "prometheus",
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

All contributions are welcome! Whether you're reporting bugs, suggesting features, or submitting code — thank you! Here's how to get involved:

▸ [Open an issue](https://github.com/freepik-company/prometheus-mcp/issues/new) to report bugs or request features

▸ [Submit a pull request](https://github.com/freepik-company/prometheus-mcp/pulls) to contribute improvements


## 📄 License

Prometheus MCP is licensed under the [Apache 2.0 License](./LICENSE).
