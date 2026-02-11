package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleToolPrometheusListMetrics handles listing available metrics
func (tm *ToolsManager) HandleToolPrometheusListMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tm.dependencies.HandlersManager.PrometheusClient == nil {
		return mcp.NewToolResultError("Prometheus client not initialized"), nil
	}

	// Parse arguments
	var args struct {
		Query string `json:"query,omitempty"`
		OrgID string `json:"org_id,omitempty"`
	}

	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments: " + err.Error()), nil
	}
	if err = json.Unmarshal(argsBytes, &args); err != nil {
		return mcp.NewToolResultError("failed to parse arguments: " + err.Error()), nil
	}

	// Validate glob pattern if provided
	if args.Query != "" {
		if _, err := filepath.Match(args.Query, ""); err != nil {
			return mcp.NewToolResultError("invalid glob pattern: " + err.Error()), nil
		}
	}

	// Get label values for __name__ which contains all metric names
	// Add org_id to context if provided
	if args.OrgID != "" {
		ctx = context.WithValue(ctx, "org_id", args.OrgID)
	}
	
	metricNames, warnings, err := tm.dependencies.HandlersManager.PrometheusClient.LabelValues(ctx, "__name__", []string{}, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return mcp.NewToolResultError("failed to fetch metrics list: " + err.Error()), nil
	}

	if len(warnings) > 0 {
		tm.dependencies.AppCtx.Logger.Warn("Prometheus list metrics warnings", "warnings", warnings)
	}

	// Format the result, applying filter if query is provided
	var result []string
	for _, name := range metricNames {
		if args.Query == "" {
			result = append(result, string(name))
		} else if matched, _ := filepath.Match(args.Query, string(name)); matched {
			result = append(result, string(name))
		}
	}

	// Convert to JSON for better formatting
	resultTOON, err := gotoon.Encode(map[string]interface{}{
		"total_metrics": len(result),
		"metrics":       result,
	})
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Available Prometheus Metrics:\n\n%s", resultTOON)), nil
}
