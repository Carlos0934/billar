package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/mark3labs/mcp-go/mcp"
)

// timeEntryServiceStub implements TimeEntryServiceProvider for testing.
type timeEntryServiceStub struct {
	recordArg       *app.RecordTimeEntryCommand
	recordRes       app.TimeEntryDTO
	recordErr       error
	getID           string
	getRes          app.TimeEntryDTO
	getErr          error
	updateArg       *app.UpdateTimeEntryCommand
	updateRes       app.TimeEntryDTO
	updateErr       error
	deleteID        string
	deleteErr       error
	listProfileID   string
	listRes         []app.TimeEntryDTO
	listErr         error
	listUnbilledID  string
	listUnbilledRes []app.TimeEntryDTO
	listUnbilledErr error
}

func (s *timeEntryServiceStub) Record(ctx context.Context, cmd app.RecordTimeEntryCommand) (app.TimeEntryDTO, error) {
	_ = ctx
	s.recordArg = &cmd
	return s.recordRes, s.recordErr
}

func (s *timeEntryServiceStub) Get(ctx context.Context, id string) (app.TimeEntryDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *timeEntryServiceStub) UpdateEntry(ctx context.Context, cmd app.UpdateTimeEntryCommand) (app.TimeEntryDTO, error) {
	_ = ctx
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *timeEntryServiceStub) DeleteEntry(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func (s *timeEntryServiceStub) ListByCustomerProfile(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error) {
	_ = ctx
	s.listProfileID = customerID
	return s.listRes, s.listErr
}

func (s *timeEntryServiceStub) ListUnbilled(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error) {
	_ = ctx
	s.listUnbilledID = customerID
	return s.listUnbilledRes, s.listUnbilledErr
}

// -- time_entry.record --

func TestTimeEntryRecordToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *timeEntryServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "records time entry successfully",
			service: &timeEntryServiceStub{
				recordRes: app.TimeEntryDTO{
					ID:                 "te_abc",
					CustomerProfileID:  "cus_1",
					ServiceAgreementID: "sa_1",
					Description:        "Write tests",
					Hours:              10000,
					Billable:           true,
				},
			},
			arguments: map[string]any{
				"customer_profile_id":  "cus_1",
				"service_agreement_id": "sa_1",
				"description":          "Write tests",
				"hours":                float64(10000),
				"billable":             true,
				"date":                 "2026-04-10T00:00:00Z",
			},
			wantResult: "te_abc",
		},
		{
			name:    "returns error when customer_profile_id is empty",
			service: &timeEntryServiceStub{},
			arguments: map[string]any{
				"description": "Work",
				"hours":       float64(10000),
				"date":        "2026-04-10T00:00:00Z",
			},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
		{
			name:    "returns tool error when service is nil",
			service: nil,
			arguments: map[string]any{
				"customer_profile_id": "cus_1",
			},
			wantErr:       true,
			wantErrSubstr: "time entry service is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc TimeEntryServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := timeEntryRecordTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "time_entry.record", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- time_entry.get --

func TestTimeEntryGetToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *timeEntryServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "gets time entry successfully",
			service: &timeEntryServiceStub{
				getRes: app.TimeEntryDTO{ID: "te_abc", Description: "Work", Hours: 10000},
			},
			arguments:  map[string]any{"id": "te_abc"},
			wantResult: "te_abc",
		},
		{
			name:          "returns error when id is empty",
			service:       &timeEntryServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name: "propagates not found error",
			service: &timeEntryServiceStub{
				getErr: errors.New("time entry not found"),
			},
			arguments:     map[string]any{"id": "te_nonexistent"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc TimeEntryServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := timeEntryGetTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "time_entry.get", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- time_entry.update --

func TestTimeEntryUpdateToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *timeEntryServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "updates time entry successfully",
			service: &timeEntryServiceStub{
				updateRes: app.TimeEntryDTO{ID: "te_abc", Description: "Updated", Hours: 20000},
			},
			arguments: map[string]any{
				"id":          "te_abc",
				"description": "Updated",
				"hours":       float64(20000),
			},
			wantResult: "te_abc",
		},
		{
			name:          "returns error when id is empty",
			service:       &timeEntryServiceStub{},
			arguments:     map[string]any{"description": "x", "hours": float64(1)},
			wantErr:       true,
			wantErrSubstr: "id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc TimeEntryServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := timeEntryUpdateTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "time_entry.update", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- time_entry.delete --

func TestTimeEntryDeleteToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *timeEntryServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name:       "deletes time entry successfully",
			service:    &timeEntryServiceStub{},
			arguments:  map[string]any{"id": "te_abc"},
			wantResult: "te_abc",
		},
		{
			name:          "returns error when id is empty",
			service:       &timeEntryServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "id",
		},
		{
			name: "propagates service error",
			service: &timeEntryServiceStub{
				deleteErr: errors.New("time entry is locked"),
			},
			arguments:     map[string]any{"id": "te_locked"},
			wantErr:       true,
			wantErrSubstr: "locked",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc TimeEntryServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := timeEntryDeleteTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "time_entry.delete", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- time_entry.list_by_customer_profile --

func TestTimeEntryListToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *timeEntryServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "lists time entries successfully",
			service: &timeEntryServiceStub{
				listRes: []app.TimeEntryDTO{
					{ID: "te_001", Description: "Task A", Hours: 5000},
				},
			},
			arguments:  map[string]any{"customer_profile_id": "cus_1"},
			wantResult: "te_001",
		},
		{
			name:          "returns error when customer_profile_id is empty",
			service:       &timeEntryServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc TimeEntryServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := timeEntryListTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "time_entry.list_by_customer_profile", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}

// -- time_entry.list_unbilled --

func TestTimeEntryListUnbilledToolHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *timeEntryServiceStub
		arguments     map[string]any
		wantErr       bool
		wantErrSubstr string
		wantResult    string
	}{
		{
			name: "lists unbilled time entries successfully",
			service: &timeEntryServiceStub{
				listUnbilledRes: []app.TimeEntryDTO{
					{ID: "te_ub1", Description: "Unbilled work", Hours: 6000},
				},
			},
			arguments:  map[string]any{"customer_profile_id": "cus_1"},
			wantResult: "te_ub1",
		},
		{
			name:          "returns error when customer_profile_id is empty",
			service:       &timeEntryServiceStub{},
			arguments:     map[string]any{},
			wantErr:       true,
			wantErrSubstr: "customer_profile_id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var svc TimeEntryServiceProvider
			if tc.service != nil {
				svc = tc.service
			}
			_, handler := timeEntryListUnbilledTool(svc, NewIngressGuard(nil), nil)
			result, err := handler(context.Background(), mcp.CallToolRequest{
				Params: mcp.CallToolParams{Name: "time_entry.list_unbilled", Arguments: tc.arguments},
			})
			if err != nil {
				t.Fatalf("handler error = %v", err)
			}

			if tc.wantErr {
				if result == nil || !result.IsError {
					t.Fatalf("handler result = %+v, want error result", result)
				}
				if tc.wantErrSubstr != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr) {
					t.Fatalf("handler error = %q, want substring %q", mcp.GetTextFromContent(result.Content[0]), tc.wantErrSubstr)
				}
				return
			}
			if result == nil || result.IsError {
				t.Fatalf("handler result = %+v, want success result", result)
			}
			if tc.wantResult != "" && !strings.Contains(mcp.GetTextFromContent(result.Content[0]), tc.wantResult) {
				t.Fatalf("handler result = %q, want contains %q", mcp.GetTextFromContent(result.Content[0]), tc.wantResult)
			}
		})
	}
}
