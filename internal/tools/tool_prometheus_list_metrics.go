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

func (tm *ToolsManager) HandleToolPrometheusListMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if tm.dependencies.HandlersManager.PrometheusClient == nil {
		return mcp.NewToolResultError("Prometheus client not initialized"), nil
	}

	var args struct {
		Query  string `json:"query,omitempty"`
		OrgID  string `json:"org_id,omitempty"`
		Limit  int    `json:"limit,omitempty"`
		Offset int    `json:"offset,omitempty"`
	}

	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments: " + err.Error()), nil
	}
	if err = json.Unmarshal(argsBytes, &args); err != nil {
		return mcp.NewToolResultError("failed to parse arguments: " + err.Error()), nil
	}

	if args.Limit <= 0 {
		args.Limit = defaultMetricsLimit
	}
	if args.Offset < 0 {
		args.Offset = 0
	}

	if args.Query != "" {
		if _, err := filepath.Match(args.Query, ""); err != nil {
			return mcp.NewToolResultError("invalid glob pattern: " + err.Error()), nil
		}
	}

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

	var filtered []string
	for _, name := range metricNames {
		if args.Query == "" {
			filtered = append(filtered, string(name))
		} else if matched, _ := filepath.Match(args.Query, string(name)); matched {
			filtered = append(filtered, string(name))
		}
	}

	totalFiltered := len(filtered)
	start := args.Offset
	end := args.Offset + args.Limit

	if start > totalFiltered {
		start = totalFiltered
	}
	if end > totalFiltered {
		end = totalFiltered
	}

	paginatedResult := filtered[start:end]
	hasMore := end < totalFiltered

	resultTOON, err := gotoon.Encode(map[string]interface{}{
		"total_metrics": totalFiltered,
		"returned":      len(paginatedResult),
		"offset":        args.Offset,
		"limit":         args.Limit,
		"has_more":      hasMore,
		"metrics":       paginatedResult,
	})
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Available Prometheus Metrics:\n\n%s", resultTOON)), nil
}
