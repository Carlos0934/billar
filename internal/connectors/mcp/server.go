package mcp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Carlos0934/billar/internal/app"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

type Server struct {
	server    *mcpsrv.MCPServer
	guard     IngressGuard
	toolNames []string
}

type CustomerListProvider interface {
	List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerDTO], error)
}

type CustomerServiceProvider interface {
	CustomerListProvider
	Create(ctx context.Context, cmd app.CreateCustomerCommand) (app.CustomerDTO, error)
	Update(ctx context.Context, id string, cmd app.PatchCustomerCommand) (app.CustomerDTO, error)
	Delete(ctx context.Context, id string) error
}

func NewServer(sessionService app.SessionService, customerService CustomerServiceProvider, guard IngressGuard, logger *slog.Logger) *Server {
	mcpServer := mcpsrv.NewMCPServer(
		"Billar MCP Session Surface",
		"1.0.0",
		mcpsrv.WithToolCapabilities(true),
		mcpsrv.WithRecovery(),
	)

	toolNames := registerTools(mcpServer, sessionService, customerService, guard, logger)

	return &Server{
		server:    mcpServer,
		guard:     guard,
		toolNames: toolNames,
	}
}

func (s *Server) ToolNames() []string {
	if s == nil {
		return nil
	}

	return append([]string(nil), s.toolNames...)
}

func (s *Server) ServeStdio() error {
	return mcpsrv.ServeStdio(s.server)
}

func (s *Server) HTTPHandler() http.Handler {
	if s == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		})
	}

	return mcpsrv.NewStreamableHTTPServer(
		s.server,
		mcpsrv.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			identity, ok, err := (app.ContextIdentitySource{}).CurrentIdentity(r.Context())
			if err != nil || !ok {
				return ctx
			}
			return app.WithAuthenticatedIdentity(ctx, identity)
		}),
	)
}
