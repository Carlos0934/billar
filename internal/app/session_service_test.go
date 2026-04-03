package app

import (
	"context"
	"reflect"
	"testing"
)

type sessionServiceStub struct{}

func (sessionServiceStub) StartLogin(context.Context) (LoginIntentDTO, error) {
	return LoginIntentDTO{LoginURL: "https://login.example"}, nil
}

func (sessionServiceStub) Status(context.Context) (SessionStatusDTO, error) {
	return SessionStatusDTO{
		Status:        "active",
		Email:         "user@example.com",
		EmailVerified: true,
		Subject:       "subject-123",
		Issuer:        "https://issuer.example",
	}, nil
}

func (sessionServiceStub) Logout(context.Context) (LogoutDTO, error) {
	return LogoutDTO{Message: "logged out"}, nil
}

func TestSessionServiceInterfaceSatisfaction(t *testing.T) {
	t.Parallel()

	var svc SessionService = sessionServiceStub{}

	login, err := svc.StartLogin(context.Background())
	if err != nil {
		t.Fatalf("StartLogin() error = %v", err)
	}
	if login.LoginURL != "https://login.example" {
		t.Fatalf("StartLogin() = %+v, want login URL", login)
	}

	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Status != "active" || status.Email != "user@example.com" || !status.EmailVerified {
		t.Fatalf("Status() = %+v, want active identity", status)
	}

	logout, err := svc.Logout(context.Background())
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if logout.Message != "logged out" {
		t.Fatalf("Logout() = %+v, want message", logout)
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
			name:   "login intent",
			typeOf: LoginIntentDTO{},
			wantTags: map[string]string{
				"LoginURL": `json:"login_url" toon:"login_url"`,
			},
		},
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
		{
			name:   "logout",
			typeOf: LogoutDTO{},
			wantTags: map[string]string{
				"Message": `json:"message" toon:"message"`,
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
