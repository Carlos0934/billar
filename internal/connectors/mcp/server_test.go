package mcp

import (
	"reflect"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestNewServerRegistersSessionTools(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		statusDTO: app.SessionStatusDTO{Status: "unauthenticated"},
	}

	server := NewServer(service, &customerWriteServiceStub{}, NewIngressGuard(nil), nil)
	want := []string{"session.status", "customer.list", "customer.create", "customer.update", "customer.delete"}
	if got := server.ToolNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToolNames() = %v, want %v", got, want)
	}
}
