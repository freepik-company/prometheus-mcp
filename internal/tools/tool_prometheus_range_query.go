package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/alpkeskin/gotoon"
	"github.com/mark3labs/mcp-go/mcp"
)

func (tm *ToolsManager) HandleToolRangeQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Backend string `json:"backend,omitempty"`
		Query   string `json:"query"`
		Start   string `json:"start"`
		End     string `json:"end"`
		Step    string `json:"step,omitempty"`
		OrgID   string `json:"org_id,omitempty"`
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

	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}
	if args.Start == "" {
		return mcp.NewToolResultError("start parameter is required"), nil
	}
	if args.End == "" {
		return mcp.NewToolResultError("end parameter is required"), nil
	}

	startTime, err := time.Parse(time.RFC3339, args.Start)
	if err != nil {
		return mcp.NewToolResultError("invalid start time format, use RFC3339: " + err.Error()), nil
	}

	var endTime time.Time
	endTime, err = time.Parse(time.RFC3339, args.End)
	if err != nil {
		return mcp.NewToolResultError("invalid end time format, use RFC3339: " + err.Error()), nil
	}

	step := time.Minute
	if args.Step != "" {
		step, err = time.ParseDuration(args.Step)
		if err != nil {
			return mcp.NewToolResultError("invalid step duration: " + err.Error()), nil
		}
	}

	result, err := tm.dependencies.HandlersManager.QueryRange(ctx, backendName, args.Query, startTime, endTime, step, args.OrgID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to execute range query on backend %q: %s", backendName, err.Error())), nil
	}

	resultTOON, err := gotoon.Encode(result)
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Range Query Results [%s]:\n\nQuery: %s\nStart: %s\nEnd: %s\nStep: %s\n\nResults:\n%s",
		backendName, args.Query, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), step.String(), resultTOON)), nil
}
