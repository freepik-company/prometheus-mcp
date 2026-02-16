package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/mark3labs/mcp-go/mcp"
)

// HandleToolPMMRangeQuery handles range PMM queries (VictoriaMetrics/PromQL compatible)
func (tm *ToolsManager) HandleToolPMMRangeQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Query string `json:"query"`
		Start string `json:"start"`
		End   string `json:"end"`
		Step  string `json:"step,omitempty"`
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
	if args.Start == "" {
		return mcp.NewToolResultError("start parameter is required"), nil
	}
	if args.End == "" {
		return mcp.NewToolResultError("end parameter is required"), nil
	}

	// Parse timestamps
	startTime, err := time.Parse(time.RFC3339, args.Start)
	if err != nil {
		return mcp.NewToolResultError("invalid start time format, use RFC3339: " + err.Error()), nil
	}

	var endTime time.Time
	endTime, err = time.Parse(time.RFC3339, args.End)
	if err != nil {
		return mcp.NewToolResultError("invalid end time format, use RFC3339: " + err.Error()), nil
	}

	// Parse step duration, default to 1 minute
	step := 1 * time.Minute
	if args.Step != "" {
		step, err = time.ParseDuration(args.Step)
		if err != nil {
			return mcp.NewToolResultError("invalid step duration: " + err.Error()), nil
		}
	}

	// Execute range query against PMM
	result, err := tm.dependencies.HandlersManager.QueryRangePMM(ctx, args.Query, startTime, endTime, step)
	if err != nil {
		return mcp.NewToolResultError("failed to execute PMM range query: " + err.Error()), nil
	}

	// Convert result to JSON
	resultTOON, err := gotoon.Encode(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("PMM Range Query Results:\n\nQuery: %s\nStart: %s\nEnd: %s\nStep: %s\n\nResults:\n%s",
		args.Query, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), step.String(), resultTOON)), nil
}
