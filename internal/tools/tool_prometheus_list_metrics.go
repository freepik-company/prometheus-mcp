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

const defaultMetricsLimit = 100

func (tm *ToolsManager) HandleToolListMetrics(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Backend string `json:"backend,omitempty"`
		Query   string `json:"query,omitempty"`
		OrgID   string `json:"org_id,omitempty"`
		Limit   int    `json:"limit,omitempty"`
		Offset  int    `json:"offset,omitempty"`
	}

	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments: " + err.Error()), nil
	}
	if err = json.Unmarshal(argsBytes, &args); err != nil {
		return mcp.NewToolResultError("failed to parse arguments: " + err.Error()), nil
	}

	backendName, err := tm.resolveBackend(args.Backend)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tm.warnIfOrgIDIgnored(backendName, args.OrgID)

	client, err := tm.dependencies.HandlersManager.GetClient(backendName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
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

	metricNames, warnings, err := client.LabelValues(ctx, "__name__", []string{}, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to fetch metrics list from backend %q: %s", backendName, err.Error())), nil
	}

	if len(warnings) > 0 {
		tm.dependencies.AppCtx.Logger.Warn("List metrics warnings", "backend", backendName, "warnings", warnings)
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

	return mcp.NewToolResultText(fmt.Sprintf("Available Metrics [%s]:\n\n%s", backendName, resultTOON)), nil
}
