package app

import (
	"context"
	"reflect"
	"testing"
)

type sessionServiceStub struct{}

func (sessionServiceStub) Status(context.Context) (SessionStatusDTO, error) {
	return SessionStatusDTO{
		Status:        "active",
		Email:         "user@example.com",
		EmailVerified: true,
		Subject:       "subject-123",
		Issuer:        "https://issuer.example",
	}, nil
}

func TestSessionServiceInterfaceSatisfaction(t *testing.T) {
	t.Parallel()

	var svc SessionService = sessionServiceStub{}

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Status != "active" || status.Email != "user@example.com" || !status.EmailVerified {
		t.Fatalf("Status() = %+v, want active identity", status)
	}
}

func TestSessionDTOFieldTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typeOf   any
		wantTags map[string]string
	}{
		{
			name:   "session status",
			typeOf: SessionStatusDTO{},
			wantTags: map[string]string{
				"Status":        `json:"status" toon:"status"`,
				"Email":         `json:"email,omitempty" toon:"email,omitempty"`,
				"EmailVerified": `json:"email_verified,omitempty" toon:"email_verified,omitempty"`,
				"Subject":       `json:"subject,omitempty" toon:"subject,omitempty"`,
				"Issuer":        `json:"issuer,omitempty" toon:"issuer,omitempty"`,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			typ := reflect.TypeOf(tc.typeOf)
			for field, want := range tc.wantTags {
				got, ok := typ.FieldByName(field)
				if !ok {
					t.Fatalf("field %q not found", field)
				}
				if got.Tag != reflect.StructTag(want) {
					t.Fatalf("%s tag = %q, want %q", field, got.Tag, want)
				}
			}
		})
	}
}
