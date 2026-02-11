package tools

import (
	"prometheus-mcp/internal/globals"
	"prometheus-mcp/internal/handlers"
	"prometheus-mcp/internal/middlewares"

	//
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

func (tm *ToolsManager) AddTools() {

	// 1. Prometheus query tool
	tool := mcp.NewTool("prometheus_query",
		mcp.WithDescription("Execute a PromQL query against Prometheus"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The PromQL query to execute"),
		),
		mcp.WithString("time",
			mcp.Description("Timestamp for the query (RFC3339 format). If not provided, uses current time"),
		),
		mcp.WithString("org_id",
			mcp.Description("Optional tenant ID for multi-tenant Prometheus/Mimir (X-Scope-OrgId header). If not provided, uses the default tenant from config"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolPrometheusQuery)

	// 2. Prometheus range query tool
	tool = mcp.NewTool("prometheus_range_query",
		mcp.WithDescription("Execute a PromQL range query against Prometheus"),
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
			mcp.Description("Optional tenant ID for multi-tenant Prometheus/Mimir (X-Scope-OrgId header). If not provided, uses the default tenant from config"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolPrometheusRangeQuery)

	// 3. Prometheus metrics list tool
	tool = mcp.NewTool("prometheus_list_metrics",
		mcp.WithDescription("List all available metrics from Prometheus"),
		mcp.WithString("query",
			mcp.Description("Optional glob pattern to filter metrics (e.g., 'redis*', '*cpu*')"),
		),
		mcp.WithString("org_id",
			mcp.Description("Optional tenant ID for multi-tenant Prometheus/Mimir (X-Scope-OrgId header). If not provided, uses the default tenant from config"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolPrometheusListMetrics)
}
