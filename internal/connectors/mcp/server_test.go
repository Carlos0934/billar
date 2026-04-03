package mcp

import (
	"reflect"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

func TestNewServerRegistersSessionTools(t *testing.T) {
	t.Parallel()

	service := &sessionServiceStub{
		startLoginDTO: app.LoginIntentDTO{LoginURL: "https://login.example"},
		statusDTO:     app.SessionStatusDTO{Status: "unauthenticated"},
		logoutDTO:     app.LogoutDTO{Message: "Logged out"},
	}

	server := NewServer(service, NewIngressGuard(DefaultAccessPolicy()))
	want := []string{"session.start_login", "session.status", "session.logout"}
	if got := server.ToolNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("ToolNames() = %v, want %v", got, want)
	}
}
