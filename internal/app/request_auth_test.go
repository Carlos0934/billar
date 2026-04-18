package app

import (
	"context"
	"errors"
	"testing"
)

type identitySourceStub struct {
	identity AuthenticatedIdentity
	ok       bool
	err      error
}

func (s identitySourceStub) CurrentIdentity(context.Context) (AuthenticatedIdentity, bool, error) {
	return s.identity, s.ok, s.err
}

func TestContextIdentitySourceCurrentIdentity(t *testing.T) {
	t.Parallel()

	identity := AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}
	got, ok, err := (ContextIdentitySource{}).CurrentIdentity(WithAuthenticatedIdentity(context.Background(), identity))
	if err != nil {
		t.Fatalf("CurrentIdentity() error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentIdentity() ok = false, want true")
	}
	if got != identity {
		t.Fatalf("CurrentIdentity() = %+v, want %+v", got, identity)
	}
}

func TestContextIdentitySourceReturnsNotOkWhenMissing(t *testing.T) {
	t.Parallel()

	_, ok, err := (ContextIdentitySource{}).CurrentIdentity(context.Background())
	if err != nil {
		t.Fatalf("CurrentIdentity() error = %v", err)
	}
	if ok {
		t.Fatal("CurrentIdentity() ok = true, want false")
	}
}

func TestStaticIdentitySourceCurrentIdentity(t *testing.T) {
	t.Parallel()

	source := NewStaticIdentitySource(AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"})

	got, ok, err := source.CurrentIdentity(context.Background())
	if err != nil {
		t.Fatalf("CurrentIdentity() error = %v", err)
	}
	if !ok {
		t.Fatal("CurrentIdentity() ok = false, want true")
	}
	if got != (AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}) {
		t.Fatalf("CurrentIdentity() = %+v", got)
	}
}

func TestRequestSessionServiceStatus(t *testing.T) {
	t.Parallel()

	svc := NewRequestSessionService(identitySourceStub{identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}, ok: true})
	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != (SessionStatusDTO{Status: "active", Email: "user@example.com", EmailVerified: true, Subject: "sub-123", Issuer: "https://issuer.example"}) {
		t.Fatalf("Status() = %+v", status)
	}
}

func TestRequestSessionServiceStatusUnauthenticated(t *testing.T) {
	t.Parallel()

	svc := NewRequestSessionService(identitySourceStub{ok: false})
	status, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.Status != "unauthenticated" {
		t.Fatalf("Status().Status = %q, want %q", status.Status, "unauthenticated")
	}
}

// TestExportedAuthErrorsAreDistinct verifies that the exported sentinel errors are
// accessible to middleware/connectors and are distinct values (not aliases of each other).
func TestExportedAuthErrorsAreDistinct(t *testing.T) {
	t.Parallel()

	// Verify each sentinel wraps itself correctly via the errors package.
	var missingErr error = ErrMissingBearerToken
	if !errors.Is(missingErr, ErrMissingBearerToken) {
		t.Fatal("ErrMissingBearerToken: errors.Is check failed — sentinel not reachable as error type")
	}

	var invalidErr error = ErrInvalidBearerToken
	if !errors.Is(invalidErr, ErrInvalidBearerToken) {
		t.Fatal("ErrInvalidBearerToken: errors.Is check failed — sentinel not reachable as error type")
	}

	// The two sentinels must NOT be equal — callers differentiate them.
	if errors.Is(ErrMissingBearerToken, ErrInvalidBearerToken) {
		t.Fatal("ErrMissingBearerToken and ErrInvalidBearerToken must be distinct error values")
	}
	if errors.Is(ErrInvalidBearerToken, ErrMissingBearerToken) {
		t.Fatal("ErrInvalidBearerToken and ErrMissingBearerToken must be distinct error values")
	}
}
