package mcp

import (
	"github.com/Carlos0934/billar/internal/app"
	mcpsrv "github.com/mark3labs/mcp-go/server"
)

type Server struct {
	server    *mcpsrv.MCPServer
	guard     IngressGuard
	toolNames []string
}

func NewServer(service app.SessionService, guard IngressGuard) *Server {
	mcpServer := mcpsrv.NewMCPServer(
		"Billar MCP Session Surface",
		"1.0.0",
		mcpsrv.WithToolCapabilities(true),
		mcpsrv.WithRecovery(),
	)

	toolNames := registerSessionTools(mcpServer, service, guard)

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
