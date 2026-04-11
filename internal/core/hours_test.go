package core

import (
	"testing"
)

// ---------------------------------------------------------------------------
// NewHours — validation table
// ---------------------------------------------------------------------------

func TestNewHours(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		val     int64
		wantErr bool
	}{
		{name: "positive 4-decimal value succeeds (1.5000 hours)", val: 15000, wantErr: false},
		{name: "positive minimum (0.0001 hours)", val: 1, wantErr: false},
		{name: "zero rejected", val: 0, wantErr: true},
		{name: "negative rejected", val: -1, wantErr: true},
		{name: "large positive succeeds", val: 80000, wantErr: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h, err := NewHours(tt.val)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewHours(%d) error = nil, want non-nil", tt.val)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewHours(%d) error = %v, want nil", tt.val, err)
			}
			if int64(h) != tt.val {
				t.Fatalf("Hours value = %d, want %d", int64(h), tt.val)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Hours.IsPositive
// ---------------------------------------------------------------------------

func TestHours_IsPositive(t *testing.T) {
	t.Parallel()

	// A valid Hours must always be positive (invariant enforced at construction)
	h, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(15000) error = %v", err)
	}
	if !h.IsPositive() {
		t.Fatal("IsPositive() = false for a valid Hours(15000), want true")
	}
}
