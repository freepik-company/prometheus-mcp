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
	dependencies HandlersManagerDependencies
	Clients      map[string]v1.API
}

func NewHandlersManager(deps HandlersManagerDependencies) *HandlersManager {
	hm := &HandlersManager{
		dependencies: deps,
		Clients:      make(map[string]v1.API),
	}

	hm.initClients(deps)

	return hm
}

func (hm *HandlersManager) initClients(deps HandlersManagerDependencies) {
	for name, backendCfg := range deps.AppCtx.Config.Backends {
		if backendCfg.URL == "" {
			deps.AppCtx.Logger.Warn("Backend has no URL configured, skipping", "backend", name)
			continue
		}

		cfgCopy := backendCfg
		transport := &backendTransport{
			transport: &http.Transport{},
			config:    &cfgCopy,
			logger:    deps.AppCtx.Logger,
			name:      name,
		}

		client, err := prometheusapi.NewClient(prometheusapi.Config{
			Address:      backendCfg.URL,
			RoundTripper: transport,
		})
		if err != nil {
			deps.AppCtx.Logger.Error("Failed to create client", "backend", name, "error", err.Error())
			continue
		}

		hm.Clients[name] = v1.NewAPI(client)
		deps.AppCtx.Logger.Info("Backend client initialized",
			"backend", name,
			"url", backendCfg.URL,
			"auth_type", backendCfg.Auth.Type,
			"org_id", backendCfg.OrgID)
	}
}

func (hm *HandlersManager) GetClient(backendName string) (v1.API, error) {
	client, ok := hm.Clients[backendName]
	if !ok {
		return nil, fmt.Errorf("backend %q not initialized", backendName)
	}
	return client, nil
}

func (hm *HandlersManager) Query(ctx context.Context, backendName string, query string, timestamp time.Time, orgID string) (interface{}, error) {
	client, err := hm.GetClient(backendName)
	if err != nil {
		return nil, err
	}

	if orgID != "" {
		ctx = context.WithValue(ctx, "org_id", orgID)
	}

	result, warnings, err := client.Query(ctx, query, timestamp)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}

	if len(warnings) > 0 {
		hm.dependencies.AppCtx.Logger.Warn("Query warnings", "backend", backendName, "warnings", warnings)
	}

	return result, nil
}

func (hm *HandlersManager) QueryRange(ctx context.Context, backendName string, query string, startTime, endTime time.Time, step time.Duration, orgID string) (interface{}, error) {
	client, err := hm.GetClient(backendName)
	if err != nil {
		return nil, err
	}

	if orgID != "" {
		ctx = context.WithValue(ctx, "org_id", orgID)
	}

	r := v1.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	}

	result, warnings, err := client.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("error executing range query: %w", err)
	}

	if len(warnings) > 0 {
		hm.dependencies.AppCtx.Logger.Warn("Range query warnings", "backend", backendName, "warnings", warnings)
	}

	return result, nil
}

type backendTransport struct {
	transport http.RoundTripper
	config    *api.BackendConfig
	logger    *slog.Logger
	name      string
}

func (bt *backendTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqClone := req.Clone(req.Context())

	orgID := bt.config.OrgID
	if ctxOrgID := req.Context().Value("org_id"); ctxOrgID != nil {
		if id, ok := ctxOrgID.(string); ok && id != "" {
			orgID = id
			bt.logger.Debug("Using org_id from context", "backend", bt.name, "org_id", orgID)
		}
	}

	if orgID != "" {
		reqClone.Header.Set("X-Scope-OrgId", orgID)
	}

	switch bt.config.Auth.Type {
	case "basic":
		if bt.config.Auth.Username != "" && bt.config.Auth.Password != "" {
			auth := bt.config.Auth.Username + ":" + bt.config.Auth.Password
			reqClone.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
			bt.logger.Debug("Added basic auth to request", "backend", bt.name, "username", bt.config.Auth.Username)
		}
	case "token":
		if bt.config.Auth.Token != "" {
			reqClone.Header.Set("Authorization", "Bearer "+bt.config.Auth.Token)
			bt.logger.Debug("Added bearer token to request", "backend", bt.name)
		}
	}

	return bt.transport.RoundTrip(reqClone)
}
