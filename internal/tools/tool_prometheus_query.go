package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/mark3labs/mcp-go/mcp"
)

func (tm *ToolsManager) HandleToolPrometheusQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Query string `json:"query"`
		Time  string `json:"time,omitempty"`
		OrgID string `json:"org_id,omitempty"`
	}

	argsBytes, err := json.Marshal(request.Params.Arguments)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal arguments: " + err.Error()), nil
	}
	if err = json.Unmarshal(argsBytes, &args); err != nil {
		return mcp.NewToolResultError("failed to parse arguments: " + err.Error()), nil
	}

	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	var timestamp time.Time
	if args.Time != "" {
		timestamp, err = time.Parse(time.RFC3339, args.Time)
		if err != nil {
			return mcp.NewToolResultError("invalid time format, use RFC3339: " + err.Error()), nil
		}
	} else {
		timestamp = time.Now()
	}

	result, err := tm.dependencies.HandlersManager.QueryPrometheus(ctx, args.Query, timestamp, args.OrgID)
	if err != nil {
		return mcp.NewToolResultError("failed to execute Prometheus query: " + err.Error()), nil
	}

	resultTOON, err := gotoon.Encode(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Prometheus Query Results:\n\nQuery: %s\nTimestamp: %s\n\nResults:\n%s",
		args.Query, timestamp.Format(time.RFC3339), resultTOON)), nil
}
