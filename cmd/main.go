package main

import (
	"log"
	"net/http"

	"prometheus-mcp/internal/globals"
	"prometheus-mcp/internal/handlers"
	"prometheus-mcp/internal/middlewares"
	"prometheus-mcp/internal/tools"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	_ = godotenv.Load() // Load .env if exists (optional for local dev)

	appCtx, err := globals.NewApplicationContext()
	if err != nil {
		log.Fatalf("failed creating application context: %v", err.Error())
	}

	accessLogsMw := middlewares.NewAccessLogsMiddleware(middlewares.AccessLogsMiddlewareDependencies{
		AppCtx: appCtx,
	})

	jwtValidationMw, err := middlewares.NewJWTValidationMiddleware(middlewares.JWTValidationMiddlewareDependencies{
		AppCtx: appCtx,
	})
	if err != nil {
		appCtx.Logger.Info("failed starting JWT validation middleware", "error", err.Error())
	}

	mcpServer := server.NewMCPServer(
		appCtx.Config.Server.Name,
		appCtx.Config.Server.Version,
		server.WithToolCapabilities(true),
	)

	hm := handlers.NewHandlersManager(handlers.HandlersManagerDependencies{
		AppCtx: appCtx,
	})

	tm := tools.NewToolsManager(tools.ToolsManagerDependencies{
		AppCtx:          appCtx,
		McpServer:       mcpServer,
		Middlewares:     []middlewares.ToolMiddleware{},
		HandlersManager: hm,
	})
	tm.AddTools()

	switch appCtx.Config.Server.Transport.Type {
	case "http":
		// NOTE: WithHeartbeatInterval is deliberately NOT set.
		// mcp-go's heartbeat emits a JSON-RPC `ping` REQUEST over the SSE
		// stream; the client answers with a normal JSON-RPC response
		// ({"jsonrpc":"2.0","id":N,"result":{}}). streamable_http.go then
		// classifies any id-bearing, method-less message carrying a result
		// as a *sampling response*, looks for a matching pending sampling
		// request, finds none, and returns HTTP 500 to the client — once per
		// heartbeat, to every client. Verified in v0.37.0 and v0.43.2.
		// Connection keep-alive is handled at the mesh instead
		// (DestinationRule connectionPool.http.idleTimeout=3600s).
		httpServer := server.NewStreamableHTTPServer(mcpServer,
			server.WithStateLess(false))

		mux := http.NewServeMux()
		mux.Handle("/mcp", accessLogsMw.Middleware(jwtValidationMw.Middleware(httpServer)))

		if appCtx.Config.OAuthAuthorizationServer.Enabled {
			mux.Handle("/.well-known/oauth-authorization-server", accessLogsMw.Middleware(http.HandlerFunc(hm.HandleOauthAuthorizationServer)))
		}

		if appCtx.Config.OAuthProtectedResource.Enabled {
			mux.Handle("/.well-known/oauth-protected-resource", accessLogsMw.Middleware(http.HandlerFunc(hm.HandleOauthProtectedResources)))
		}

		appCtx.Logger.Info("starting StreamableHTTP server", "host", appCtx.Config.Server.Transport.HTTP.Host)
		if err := http.ListenAndServe(appCtx.Config.Server.Transport.HTTP.Host, mux); err != nil {
			log.Fatal(err)
		}

	default:
		appCtx.Logger.Info("starting stdio server")
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatal(err)
		}
	}
}
