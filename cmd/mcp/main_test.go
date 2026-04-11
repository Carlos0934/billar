package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	mcpconnector "github.com/Carlos0934/billar/internal/connectors/mcp"
	"github.com/Carlos0934/billar/internal/infra/config"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func TestNewServerWiresTimeEntryTools(t *testing.T) {
	t.Parallel()

	store := mustOpenMCPStore(t)
	server, err := newServer(
		config.Config{AccessPolicy: config.AccessPolicy{AllowedEmails: []string{"integration@example.com"}}},
		"integration@example.com",
		store,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("newServer() error = %v", err)
	}

	tools := strings.Join(server.ToolNames(), ",")
	for _, want := range []string{"session.status", "time_entry.record", "time_entry.list_unbilled"} {
		if !strings.Contains(tools, want) {
			t.Fatalf("ToolNames() = %q, want %q", tools, want)
		}
	}
}

func TestNewServerRejectsDisallowedLocalAuthEmail(t *testing.T) {
	t.Parallel()

	store := mustOpenMCPStore(t)
	_, err := newServer(
		config.Config{AccessPolicy: config.AccessPolicy{AllowedEmails: []string{"allowed@example.com"}}},
		"blocked@example.com",
		store,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err == nil {
		t.Fatal("newServer() error = nil, want policy rejection")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("newServer() error = %q, want policy rejection", err)
	}
}

func TestMainWiresAndServesStdio(t *testing.T) {
	storePath := t.TempDir() + "/mcp-main.db"
	t.Setenv("BILLAR_DB_PATH", storePath)
	t.Setenv("BILLAR_LOCAL_AUTH_EMAIL", "integration@example.com")

	oldServeStdio := serveStdio
	serveStdio = func(server *mcpconnector.Server) error {
		if server == nil {
			t.Fatal("ServeStdio received nil server")
		}
		return nil
	}
	t.Cleanup(func() { serveStdio = oldServeStdio })

	main()
}

func mustOpenMCPStore(t *testing.T) *infrasqlite.Store {
	t.Helper()

	store, err := infrasqlite.Open(t.TempDir() + "/mcp-entrypoint.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}
