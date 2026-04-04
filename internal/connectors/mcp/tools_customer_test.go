package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

type contextAwareCustomerProvider struct{}

func (contextAwareCustomerProvider) List(ctx context.Context, _ app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	identity, ok, err := (app.ContextIdentitySource{}).CurrentIdentity(ctx)
	if err != nil {
		return app.ListResult[app.CustomerDTO]{}, err
	}
	if !ok || identity.Email != "user@example.com" {
		return app.ListResult[app.CustomerDTO]{}, app.ErrCustomerListAccessDenied
	}
	return app.ListResult[app.CustomerDTO]{}, nil
}

type customerListServiceStub struct {
	called bool
	query  app.ListQuery
	result app.ListResult[app.CustomerDTO]
	err    error
}

func (s *customerListServiceStub) List(ctx context.Context, query app.ListQuery) (app.ListResult[app.CustomerDTO], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func TestCustomerListToolHandlers(t *testing.T) {
	t.Parallel()

	service := &customerListServiceStub{
		result: app.ListResult[app.CustomerDTO]{
			Items: []app.CustomerDTO{{
				ID:              "cus_123",
				Type:            "company",
				LegalName:       "Acme SRL",
				TradeName:       "Acme",
				Email:           "billing@acme.example",
				Status:          "active",
				DefaultCurrency: "USD",
				CreatedAt:       "2026-04-03T10:00:00Z",
				UpdatedAt:       "2026-04-03T10:05:00Z",
			}},
			Total:    1,
			Page:     2,
			PageSize: 1,
		},
	}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := customerListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "127.0.0.1",
	}), Params: mcp.CallToolParams{Name: "customer.list", Arguments: map[string]any{
		"search":    "  Acme  ",
		"sort":      "created_at:desc",
		"page":      2,
		"page_size": 1,
	}}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if len(result.Content) != 1 {
		t.Fatalf("handler result content = %d, want 1", len(result.Content))
	}
	want := "Billar Customers\n───────────────\nPage: 2\nPage size: 1\nTotal: 1\n\n1. Acme SRL\n   Trade name: Acme\n   Type: company\n   Status: active\n   Email: billing@acme.example\n   Default currency: USD\n   Created at: 2026-04-03T10:00:00Z\n   Updated at: 2026-04-03T10:05:00Z\n"
	if got := mcp.GetTextFromContent(result.Content[0]); got != want {
		t.Fatalf("handler text = %q, want %q", got, want)
	}
	if !service.called {
		t.Fatal("List() was not called")
	}
	if service.query != (app.ListQuery{Search: "Acme", SortField: "created_at", SortDir: "desc", Page: 2, PageSize: 1}) {
		t.Fatalf("List() query = %+v", service.query)
	}
}

func TestCustomerListToolHandlersRejectIngress(t *testing.T) {
	t.Parallel()

	service := &customerListServiceStub{}
	guard := NewIngressGuard([]string{"127.0.0.1"})

	_, handler := customerListTool(service, guard, nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Header: headerWithValues(map[string]string{
		"X-Forwarded-For": "192.0.2.10",
	}), Params: mcp.CallToolParams{Name: "customer.list"}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("handler result = %+v, want error result", result)
	}
	if service.called {
		t.Fatal("List() was called for rejected request")
	}
}

func TestCustomerListToolUsesContextAuthenticatedIdentity(t *testing.T) {
	t.Parallel()

	service := contextAwareCustomerProvider{}
	_, handler := customerListTool(service, NewIngressGuard(nil), nil)
	result, err := handler(app.WithAuthenticatedIdentity(context.Background(), app.AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer.list"}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
}

func TestCustomerListToolHandlersRejectBadSort(t *testing.T) {
	t.Parallel()

	service := &customerListServiceStub{}
	_, handler := customerListTool(service, NewIngressGuard(nil), nil)
	result, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: "customer.list", Arguments: map[string]any{
		"sort": "foo:bar",
	}}})
	if err != nil {
		t.Fatalf("handler error = %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("handler result = %+v, want success result", result)
	}
	if !strings.Contains(mcp.GetTextFromContent(result.Content[0]), "Billar Customers") {
		t.Fatalf("handler text = %q", mcp.GetTextFromContent(result.Content[0]))
	}
}
