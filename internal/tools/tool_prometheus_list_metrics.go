package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleToolPrometheusListMetrics handles listing available metrics
func (tm *ToolsManager) HandleToolPrometheusListMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tm.dependencies.HandlersManager.PrometheusClient == nil {
		return mcp.NewToolResultError("Prometheus client not initialized"), nil
	}

	// Get label values for __name__ which contains all metric names
	metricNames, warnings, err := tm.dependencies.HandlersManager.PrometheusClient.LabelValues(ctx, "__name__", []string{}, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return mcp.NewToolResultError("failed to fetch metrics list: " + err.Error()), nil
	}

	if len(warnings) > 0 {
		tm.dependencies.AppCtx.Logger.Warn("Prometheus list metrics warnings", "warnings", warnings)
	}

	// Format the result
	var result []string
	for _, name := range metricNames {
		result = append(result, string(name))
	}

	// Convert to JSON for better formatting
	resultJSON, err := json.MarshalIndent(map[string]interface{}{
		"total_metrics": len(result),
		"metrics":       result,
	}, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Available Prometheus Metrics:\n\n%s", string(resultJSON))), nil
}
