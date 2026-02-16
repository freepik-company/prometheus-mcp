package handlers

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"prometheus-mcp/api"
	"prometheus-mcp/internal/globals"
	"time"

	prometheusapi "github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type HandlersManagerDependencies struct {
	AppCtx *globals.ApplicationContext
}

type HandlersManager struct {
	dependencies     HandlersManagerDependencies
	PrometheusClient v1.API
	PMMClient        v1.API
}

func NewHandlersManager(deps HandlersManagerDependencies) *HandlersManager {
	hm := &HandlersManager{
		dependencies: deps,
	}

	// 1. Initialize prometheus client for queries
	if deps.AppCtx.Config.Prometheus.URL != "" {
		// Create HTTP client with custom transport for headers and auth
		transport := &http.Transport{}
		httpClient := &http.Client{Transport: &prometheusTransport{
			transport: transport,
			config:    &deps.AppCtx.Config.Prometheus,
			logger:    deps.AppCtx.Logger,
		}}

		// Parse timeout if provided
		if deps.AppCtx.Config.Prometheus.Timeout != "" {
			if timeout, err := time.ParseDuration(deps.AppCtx.Config.Prometheus.Timeout); err == nil {
				httpClient.Timeout = timeout
			} else {
				deps.AppCtx.Logger.Warn("invalid prometheus timeout, using default", "timeout", deps.AppCtx.Config.Prometheus.Timeout, "error", err.Error())
			}
		}

		client, err := prometheusapi.NewClient(prometheusapi.Config{
			Address:      deps.AppCtx.Config.Prometheus.URL,
			RoundTripper: httpClient.Transport,
		})
		if err != nil {
			deps.AppCtx.Logger.Error("failed to create Prometheus client", "error", err.Error())
		} else {
			hm.PrometheusClient = v1.NewAPI(client)
			deps.AppCtx.Logger.Info("Prometheus client initialized successfully",
				"url", deps.AppCtx.Config.Prometheus.URL,
				"auth_type", deps.AppCtx.Config.Prometheus.Auth.Type,
				"org_id", deps.AppCtx.Config.Prometheus.OrgID)
		}
	}

	// 2. Initialize PMM client for queries (VictoriaMetrics compatible)
	if deps.AppCtx.Config.PMM.URL != "" {
		transport := &http.Transport{}
		httpClient := &http.Client{Transport: &pmmTransport{
			transport: transport,
			config:    &deps.AppCtx.Config.PMM,
			logger:    deps.AppCtx.Logger,
		}}

		// Parse timeout if provided
		if deps.AppCtx.Config.PMM.Timeout != "" {
			if timeout, err := time.ParseDuration(deps.AppCtx.Config.PMM.Timeout); err == nil {
				httpClient.Timeout = timeout
			} else {
				deps.AppCtx.Logger.Warn("invalid PMM timeout, using default", "timeout", deps.AppCtx.Config.PMM.Timeout, "error", err.Error())
			}
		}

		client, err := prometheusapi.NewClient(prometheusapi.Config{
			Address:      deps.AppCtx.Config.PMM.URL,
			RoundTripper: httpClient.Transport,
		})
		if err != nil {
			deps.AppCtx.Logger.Error("failed to create PMM client", "error", err.Error())
		} else {
			hm.PMMClient = v1.NewAPI(client)
			deps.AppCtx.Logger.Info("PMM client initialized successfully",
				"url", deps.AppCtx.Config.PMM.URL,
				"auth_type", deps.AppCtx.Config.PMM.Auth.Type)
		}
	}

	return hm
}

// QueryPrometheus executes a PromQL query against Prometheus
func (hm *HandlersManager) QueryPrometheus(ctx context.Context, query string, timestamp time.Time, orgID string) (interface{}, error) {
	if hm.PrometheusClient == nil {
		return nil, fmt.Errorf("prometheus client not initialized")
	}

	// Add org_id to context if provided for dynamic tenant override
	if orgID != "" {
		ctx = context.WithValue(ctx, "org_id", orgID)
	}

	result, warnings, err := hm.PrometheusClient.Query(ctx, query, timestamp)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}

	if len(warnings) > 0 {
		hm.dependencies.AppCtx.Logger.Warn("Prometheus query warnings", "warnings", warnings)
	}

	return result, nil
}

// QueryRangePrometheus executes a range query against Prometheus
func (hm *HandlersManager) QueryRangePrometheus(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration, orgID string) (interface{}, error) {
	if hm.PrometheusClient == nil {
		return nil, fmt.Errorf("prometheus client not initialized")
	}

	// Add org_id to context if provided for dynamic tenant override
	if orgID != "" {
		ctx = context.WithValue(ctx, "org_id", orgID)
	}

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	result, warnings, err := hm.PrometheusClient.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("error executing range query: %w", err)
	}

	if len(warnings) > 0 {
		hm.dependencies.AppCtx.Logger.Warn("Prometheus range query warnings", "warnings", warnings)
	}

	return result, nil
}

// prometheusTransport is a custom HTTP transport that adds authentication and headers
type prometheusTransport struct {
	transport http.RoundTripper
	config    *api.PrometheusConfig
	logger    *slog.Logger
}

// RoundTrip implements http.RoundTripper interface
func (pt *prometheusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqClone := req.Clone(req.Context())

	// Determine org_id: use from context if provided, otherwise use default from config
	orgID := pt.config.OrgID // Default from config
	if ctxOrgID := req.Context().Value("org_id"); ctxOrgID != nil {
		if id, ok := ctxOrgID.(string); ok && id != "" {
			orgID = id // Override with context value
			pt.logger.Debug("Using org_id from context", "org_id", orgID)
		}
	}

	// Add X-Scope-OrgId header if we have an org_id (from context or config)
	if orgID != "" {
		reqClone.Header.Set("X-Scope-OrgId", orgID)
	}

	// Add authentication based on type
	switch pt.config.Auth.Type {
	case "basic":
		if pt.config.Auth.Username != "" && pt.config.Auth.Password != "" {
			auth := pt.config.Auth.Username + ":" + pt.config.Auth.Password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			reqClone.Header.Set("Authorization", basicAuth)
			pt.logger.Debug("Added basic auth to Prometheus request", "username", pt.config.Auth.Username)
		}
	case "token":
		if pt.config.Auth.Token != "" {
			reqClone.Header.Set("Authorization", "Bearer "+pt.config.Auth.Token)
			pt.logger.Debug("Added bearer token to Prometheus request")
		}
	}

	// Use the underlying transport to make the actual request
	return pt.transport.RoundTrip(reqClone)
}

// pmmTransport is a custom HTTP transport for PMM that adds authentication
type pmmTransport struct {
	transport http.RoundTripper
	config    *api.PMMConfig
	logger    *slog.Logger
}

// RoundTrip implements http.RoundTripper interface for PMM
func (pt *pmmTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqClone := req.Clone(req.Context())

	// Add authentication based on type
	switch pt.config.Auth.Type {
	case "basic":
		if pt.config.Auth.Username != "" && pt.config.Auth.Password != "" {
			auth := pt.config.Auth.Username + ":" + pt.config.Auth.Password
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
			reqClone.Header.Set("Authorization", basicAuth)
			pt.logger.Debug("Added basic auth to PMM request", "username", pt.config.Auth.Username)
		}
	case "token":
		if pt.config.Auth.Token != "" {
			reqClone.Header.Set("Authorization", "Bearer "+pt.config.Auth.Token)
			pt.logger.Debug("Added bearer token to PMM request")
		}
	}

	// Use the underlying transport to make the actual request
	return pt.transport.RoundTrip(reqClone)
}

// QueryPMM executes a PromQL query against PMM (VictoriaMetrics)
func (hm *HandlersManager) QueryPMM(ctx context.Context, query string, timestamp time.Time) (interface{}, error) {
	if hm.PMMClient == nil {
		return nil, fmt.Errorf("PMM client not initialized")
	}

	result, warnings, err := hm.PMMClient.Query(ctx, query, timestamp)
	if err != nil {
		return nil, fmt.Errorf("error executing PMM query: %w", err)
	}

	if len(warnings) > 0 {
		hm.dependencies.AppCtx.Logger.Warn("PMM query warnings", "warnings", warnings)
	}

	return result, nil
}

// QueryRangePMM executes a range query against PMM (VictoriaMetrics)
func (hm *HandlersManager) QueryRangePMM(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration) (interface{}, error) {
	if hm.PMMClient == nil {
		return nil, fmt.Errorf("PMM client not initialized")
	}

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	result, warnings, err := hm.PMMClient.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("error executing PMM range query: %w", err)
	}

	if len(warnings) > 0 {
		hm.dependencies.AppCtx.Logger.Warn("PMM range query warnings", "warnings", warnings)
	}

	return result, nil
}
