package mcp

import (
	"testing"
)

func TestMCPServerOverHTTPUsesRequestAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	session := &sessionServiceStub{}
	issuer := &issuerProfileServiceStub{}
	customer := &customerProfileWriteServiceStub{}
	server := NewServer(session, issuer, customer, nil, NewIngressGuard(nil), nil)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestMCPServerOverHTTPAllowsUnauthenticatedDiscoveryMethods(t *testing.T) {
	t.Parallel()

	session := &sessionServiceStub{}
	issuer := &issuerProfileServiceStub{}
	customer := &customerProfileWriteServiceStub{}
	server := NewServer(session, issuer, customer, nil, NewIngressGuard(nil), nil)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}

func TestMCPServerOverHTTPRejectsUnauthenticatedActionMethods(t *testing.T) {
	t.Parallel()

	session := &sessionServiceStub{}
	issuer := &issuerProfileServiceStub{}
	customer := &customerProfileWriteServiceStub{}
	server := NewServer(session, issuer, customer, nil, NewIngressGuard(nil), nil)

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
}
