package mcp

import (
	"testing"
)

func TestNewServerRegistersSessionTools(t *testing.T) {
	t.Parallel()

	session := &sessionServiceStub{}
	legalEntity := &legalEntityWriteServiceStub{}
	issuer := &issuerProfileServiceStub{}
	customer := &customerProfileWriteServiceStub{}
	server := NewServer(session, legalEntity, issuer, customer, NewIngressGuard(nil), nil)
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	// At a minimum, session.status should be registered
	tools := server.ToolNames()
	if len(tools) < 1 {
		t.Fatalf("expected at least 1 tool, got %d", len(tools))
	}
}
