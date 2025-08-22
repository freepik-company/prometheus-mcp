package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleToolPrometheusQuery handles instant Prometheus queries
func (tm *ToolsManager) HandleToolPrometheusQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Query string `json:"query"`
		Time  string `json:"time,omitempty"`
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

	// Parse timestamp, default to now if not provided
	var timestamp time.Time
	if args.Time != "" {
		timestamp, err = time.Parse(time.RFC3339, args.Time)
		if err != nil {
			return mcp.NewToolResultError("invalid time format, use RFC3339: " + err.Error()), nil
		}
	} else {
		timestamp = time.Now()
	}

	// Execute query
	result, err := tm.dependencies.HandlersManager.QueryPrometheus(ctx, args.Query, timestamp)
	if err != nil {
		return mcp.NewToolResultError("failed to execute Prometheus query: " + err.Error()), nil
	}

	// Convert result to JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Prometheus Query Results:\n\nQuery: %s\nTimestamp: %s\n\nResults:\n%s",
		args.Query, timestamp.Format(time.RFC3339), string(resultJSON))), nil
}

// HandleToolPrometheusRangeQuery handles range Prometheus queries
func (tm *ToolsManager) HandleToolPrometheusRangeQuery(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	// Execute range query
	result, err := tm.dependencies.HandlersManager.QueryRangePrometheus(ctx, args.Query, startTime, endTime, step)
	if err != nil {
		return mcp.NewToolResultError("failed to execute Prometheus range query: " + err.Error()), nil
	}

	// Convert result to JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("failed to marshal result: " + err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Prometheus Range Query Results:\n\nQuery: %s\nStart: %s\nEnd: %s\nStep: %s\n\nResults:\n%s",
		args.Query, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), step.String(), string(resultJSON))), nil
}

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
