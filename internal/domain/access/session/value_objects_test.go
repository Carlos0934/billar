package session_test

import (
	"testing"

	"github.com/Carlos0934/billar/internal/domain/access/session"
)

func TestNewSessionID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "valid", input: "session-123", want: "session-123"},
		{name: "trims whitespace", input: "  session-123  ", want: "session-123"},
		{name: "rejects blank", input: "   ", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := session.NewSessionID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if got := id.String(); got != tt.want {
				t.Fatalf("id = %q, want %q", got, tt.want)
			}
			if id.IsZero() {
				t.Fatal("expected non-zero session id")
			}
		})
	}
}

func TestZeroSessionIDAndStatusValues(t *testing.T) {
	var zeroID session.SessionID
	if !zeroID.IsZero() {
		t.Fatal("expected zero session id to report zero")
	}

	if got := string(session.StatusLocked); got != "locked" {
		t.Fatalf("locked status = %q, want %q", got, "locked")
	}
	if got := string(session.StatusUnlocked); got != "unlocked" {
		t.Fatalf("unlocked status = %q, want %q", got, "unlocked")
	}
}
