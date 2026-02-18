package tools

import (
	"fmt"
	"sort"
	"strings"

	"prometheus-mcp/internal/globals"
	"prometheus-mcp/internal/handlers"
	"prometheus-mcp/internal/middlewares"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolsManagerDependencies struct {
	AppCtx *globals.ApplicationContext

	McpServer       *server.MCPServer
	Middlewares     []middlewares.ToolMiddleware
	HandlersManager *handlers.HandlersManager
}

type ToolsManager struct {
	dependencies ToolsManagerDependencies
}

func NewToolsManager(deps ToolsManagerDependencies) *ToolsManager {
	return &ToolsManager{
		dependencies: deps,
	}
}

func (tm *ToolsManager) resolveBackend(backendArg string) (string, error) {
	backends := tm.dependencies.AppCtx.Config.Backends
	if len(backends) == 0 {
		return "", fmt.Errorf("no backends configured")
	}
	if backendArg == "" {
		if len(backends) == 1 {
			for name := range backends {
				return name, nil
			}
		}
		return "", fmt.Errorf("backend parameter required when multiple backends are configured")
	}
	if _, ok := backends[backendArg]; !ok {
		available := tm.backendNames()
		return "", fmt.Errorf("unknown backend %q, available: [%s]", backendArg, strings.Join(available, ", "))
	}
	return backendArg, nil
}

func (tm *ToolsManager) backendNames() []string {
	names := make([]string, 0, len(tm.dependencies.AppCtx.Config.Backends))
	for name := range tm.dependencies.AppCtx.Config.Backends {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (tm *ToolsManager) buildBackendDescription() string {
	names := tm.backendNames()
	desc := fmt.Sprintf("Backend to query. Available: [%s].", strings.Join(names, ", "))
	if len(names) == 1 {
		desc += fmt.Sprintf(" Defaults to '%s' if not specified.", names[0])
	}
	return desc
}

func (tm *ToolsManager) warnIfOrgIDIgnored(backendName, orgID string) {
	if orgID == "" {
		return
	}
	cfg, ok := tm.dependencies.AppCtx.Config.Backends[backendName]
	if !ok {
		return
	}
	if len(cfg.AvailableOrgs) == 0 && cfg.OrgID == "" {
		tm.dependencies.AppCtx.Logger.Warn("org_id provided but backend has no multi-tenant configuration, header will be sent but may be ignored",
			"backend", backendName,
			"org_id", orgID,
		)
	}
}

func (tm *ToolsManager) buildOrgIDDescription() string {
	baseDesc := "Optional tenant ID for multi-tenant Prometheus/Mimir (X-Scope-OrgId header)."

	for _, cfg := range tm.dependencies.AppCtx.Config.Backends {
		if cfg.OrgID != "" {
			baseDesc += fmt.Sprintf(" Default: '%s'.", cfg.OrgID)
			break
		}
	}

	for _, cfg := range tm.dependencies.AppCtx.Config.Backends {
		if len(cfg.AvailableOrgs) > 0 {
			baseDesc += fmt.Sprintf(" Available tenants: [%s].", strings.Join(cfg.AvailableOrgs, ", "))
			break
		}
	}

	return baseDesc
}

func (tm *ToolsManager) AddTools() {
	backendDesc := tm.buildBackendDescription()
	orgIDDesc := tm.buildOrgIDDescription()

	tool := mcp.NewTool("prometheus_query",
		mcp.WithDescription("Execute a PromQL query against a metrics backend"),
		mcp.WithString("backend",
			mcp.Description(backendDesc),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The PromQL query to execute"),
		),
		mcp.WithString("time",
			mcp.Description("Timestamp for the query (RFC3339 format). If not provided, uses current time"),
		),
		mcp.WithString("org_id",
			mcp.Description(orgIDDesc),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolQuery)

	tool = mcp.NewTool("prometheus_range_query",
		mcp.WithDescription("Execute a PromQL range query against a metrics backend"),
		mcp.WithString("backend",
			mcp.Description(backendDesc),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The PromQL query to execute"),
		),
		mcp.WithString("start",
			mcp.Required(),
			mcp.Description("Start time for the range query (RFC3339 format)"),
		),
		mcp.WithString("end",
			mcp.Required(),
			mcp.Description("End time for the range query (RFC3339 format)"),
		),
		mcp.WithString("step",
			mcp.Description("Step duration for the range query (e.g., '30s', '1m', '5m'). Defaults to '1m'"),
		),
		mcp.WithString("org_id",
			mcp.Description(orgIDDesc),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolRangeQuery)

	tool = mcp.NewTool("prometheus_list_metrics",
		mcp.WithDescription("List all available metrics from a metrics backend"),
		mcp.WithString("backend",
			mcp.Description(backendDesc),
		),
		mcp.WithString("query",
			mcp.Description("Optional glob pattern to filter metrics (e.g., 'redis*', '*cpu*')"),
		),
		mcp.WithString("org_id",
			mcp.Description(orgIDDesc),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of metrics to return. Defaults to 100."),
		),
		mcp.WithNumber("offset",
			mcp.Description("Number of metrics to skip for pagination. Defaults to 0."),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolListMetrics)
}
