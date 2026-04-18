package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type stubInvoiceService struct {
	draftArg        *app.CreateDraftFromUnbilledCommand
	draftRes        app.InvoiceDTO
	draftErr        error
	issueArg        *app.IssueInvoiceCommand
	issueRes        app.InvoiceDTO
	issueErr        error
	discardID       string
	discardRes      app.DiscardResult
	discardErr      error
	getInvoiceID    string
	getInvoiceRes   app.InvoiceDTO
	getInvoiceErr   error
	listInvoicesRes []app.InvoiceSummaryDTO
	listInvoicesErr error
}

func (s *stubInvoiceService) CreateDraftFromUnbilled(ctx context.Context, cmd app.CreateDraftFromUnbilledCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.draftArg = &cmd
	return s.draftRes, s.draftErr
}

func (s *stubInvoiceService) IssueDraft(ctx context.Context, cmd app.IssueInvoiceCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.issueArg = &cmd
	return s.issueRes, s.issueErr
}

func (s *stubInvoiceService) Discard(ctx context.Context, id string) (app.DiscardResult, error) {
	_ = ctx
	s.discardID = id
	return s.discardRes, s.discardErr
}

func (s *stubInvoiceService) GetInvoice(ctx context.Context, id string) (app.InvoiceDTO, error) {
	_ = ctx
	s.getInvoiceID = id
	return s.getInvoiceRes, s.getInvoiceErr
}

func (s *stubInvoiceService) ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]app.InvoiceSummaryDTO, error) {
	_ = ctx
	return s.listInvoicesRes, s.listInvoicesErr
}

func newTestInvoiceCommand(svc InvoiceServiceProvider) Command {
	return NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, svc, false)
}

func TestInvoiceDraftCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubInvoiceService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantContain   string
	}{
		{
			name: "creates draft successfully",
			service: &stubInvoiceService{
				draftRes: app.InvoiceDTO{ID: "inv_001", CustomerID: "cus_1", Status: "draft", IsDraft: true},
			},
			args:        []string{"invoice", "draft", "--customer-id", "cus_1"},
			wantContain: "inv_001",
		},
		{
			name:          "missing customer-id",
			service:       &stubInvoiceService{},
			args:          []string{"invoice", "draft"},
			wantErr:       true,
			wantErrSubstr: "--customer-id is required",
		},
		{
			name: "propagates service error",
			service: &stubInvoiceService{
				draftErr: errors.New("no unbilled time entries"),
			},
			args:          []string{"invoice", "draft", "--customer-id", "cus_1"},
			wantErr:       true,
			wantErrSubstr: "no unbilled time entries",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newTestInvoiceCommand(tc.service)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Fatalf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestInvoiceIssueCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubInvoiceService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantContain   string
	}{
		{
			name: "issues invoice successfully",
			service: &stubInvoiceService{
				issueRes: app.InvoiceDTO{ID: "inv_001", InvoiceNumber: "INV-2026-0001", Status: "issued", IsIssued: true},
			},
			args:        []string{"invoice", "issue", "--id", "inv_001"},
			wantContain: "INV-2026-0001",
		},
		{
			name:          "missing id",
			service:       &stubInvoiceService{},
			args:          []string{"invoice", "issue"},
			wantErr:       true,
			wantErrSubstr: "--id is required",
		},
		{
			name: "propagates service error",
			service: &stubInvoiceService{
				issueErr: errors.New("invoice is not draft"),
			},
			args:          []string{"invoice", "issue", "--id", "inv_001"},
			wantErr:       true,
			wantErrSubstr: "invoice is not draft",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newTestInvoiceCommand(tc.service)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Fatalf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestInvoiceDiscardCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		service       *stubInvoiceService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantContain   string
	}{
		{
			name: "hard-discard draft successfully",
			service: &stubInvoiceService{
				discardRes: app.DiscardResult{WasSoftDiscard: false},
			},
			args:        []string{"invoice", "discard", "--id", "inv_001"},
			wantContain: "deleted",
		},
		{
			name: "soft-discard issued shows warning",
			service: &stubInvoiceService{
				discardRes: app.DiscardResult{WasSoftDiscard: true, InvoiceNumber: "INV-2026-0001"},
			},
			args:        []string{"invoice", "discard", "--id", "inv_001"},
			wantContain: "INV-2026-0001",
		},
		{
			name:          "missing id",
			service:       &stubInvoiceService{},
			args:          []string{"invoice", "discard"},
			wantErr:       true,
			wantErrSubstr: "--id is required",
		},
		{
			name: "propagates service error",
			service: &stubInvoiceService{
				discardErr: errors.New("invoice is already discarded"),
			},
			args:          []string{"invoice", "discard", "--id", "inv_001"},
			wantErr:       true,
			wantErrSubstr: "already discarded",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newTestInvoiceCommand(tc.service)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Fatalf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestInvoiceCommand_RejectsNilService(t *testing.T) {
	t.Parallel()

	cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
	err := cmd.Run(context.Background(), []string{"invoice", "draft", "--customer-id", "cus_1"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "invoice service is required") {
		t.Fatalf("Run() error = %v, want invoice service is required", err)
	}
}

func TestInvoiceCommand_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	cmd := newTestInvoiceCommand(&stubInvoiceService{})
	err := cmd.Run(context.Background(), []string{"invoice", "unknown"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("Run() error = %v, want unknown command", err)
	}
}

func TestInvoiceShowCommand(t *testing.T) {
	t.Parallel()

	wantShowExact := "Invoice\n" +
		"───────\n" +
		"ID: inv_001\n" +
		"Number: INV-001\n" +
		"Customer: cus_1\n" +
		"Status: issued\n" +
		"Currency: USD\n" +
		"\n" +
		"Lines\n" +
		"─────\n" +
		"Description                Qty(min)   Rate         Total\n" +
		"Consulting                 90         10000   USD  15000   USD\n" +
		"\n" +
		"Totals\n" +
		"──────\n" +
		"Subtotal: 15000 USD\n" +
		"Grand Total: 15000 USD\n"

	tests := []struct {
		name          string
		service       *stubInvoiceService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantExact     string
		wantContains  []string
	}{
		{
			name: "show invoice text layout exact",
			service: &stubInvoiceService{
				getInvoiceRes: app.InvoiceDTO{
					ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD",
					Lines: []app.InvoiceLineDTO{
						{Description: "Consulting", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"},
					},
					Subtotal: 15000, GrandTotal: 15000,
				},
			},
			args:      []string{"invoice", "show", "--id", "inv_001"},
			wantExact: wantShowExact,
		},
		{
			name:          "missing id",
			service:       &stubInvoiceService{},
			args:          []string{"invoice", "show"},
			wantErr:       true,
			wantErrSubstr: "--id is required",
		},
		{
			name:          "service error",
			service:       &stubInvoiceService{getInvoiceErr: errors.New("not found")},
			args:          []string{"invoice", "show", "--id", "inv_999"},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newTestInvoiceCommand(tc.service)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			got := out.String()
			if tc.wantExact != "" {
				if got != tc.wantExact {
					t.Fatalf("output mismatch:\ngot:\n%s\nwant:\n%s", got, tc.wantExact)
				}
				return
			}
			for _, want := range tc.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\ngot:\n%s", want, got)
				}
			}
		})
	}
}

func TestInvoiceListCommand(t *testing.T) {
	t.Parallel()

	wantListExact := "Invoices\n" +
		"────────\n" +
		"Customer: cus_1\n" +
		"Count: 2\n" +
		"\n" +
		"Number           Status       Currency   Total\n" +
		"INV-001          issued       USD        15000\n" +
		"—                draft        USD        3000\n"

	wantFilterExact := "Invoices\n" +
		"────────\n" +
		"Customer: cus_1\n" +
		"Status: draft\n" +
		"Count: 1\n" +
		"\n" +
		"Number           Status       Currency   Total\n" +
		"—                draft        USD        3000\n"

	tests := []struct {
		name          string
		service       *stubInvoiceService
		args          []string
		wantErr       bool
		wantErrSubstr string
		wantExact     string
	}{
		{
			name: "list with results exact",
			service: &stubInvoiceService{
				listInvoicesRes: []app.InvoiceSummaryDTO{
					{ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", GrandTotal: 15000},
					{ID: "inv_002", InvoiceNumber: "", CustomerID: "cus_1", Status: "draft", Currency: "USD", GrandTotal: 3000},
				},
			},
			args:      []string{"invoice", "list", "--customer-id", "cus_1"},
			wantExact: wantListExact,
		},
		{
			name: "list with status filter exact",
			service: &stubInvoiceService{
				listInvoicesRes: []app.InvoiceSummaryDTO{
					{ID: "inv_002", InvoiceNumber: "", CustomerID: "cus_1", Status: "draft", Currency: "USD", GrandTotal: 3000},
				},
			},
			args:      []string{"invoice", "list", "--customer-id", "cus_1", "--status", "draft"},
			wantExact: wantFilterExact,
		},
		{
			name:      "empty list",
			service:   &stubInvoiceService{listInvoicesRes: nil},
			args:      []string{"invoice", "list", "--customer-id", "cus_999"},
			wantExact: "No invoices found.\n",
		},
		{
			name:          "missing customer-id",
			service:       &stubInvoiceService{},
			args:          []string{"invoice", "list"},
			wantErr:       true,
			wantErrSubstr: "--customer-id is required",
		},
		{
			name:          "service error",
			service:       &stubInvoiceService{listInvoicesErr: errors.New("store failure")},
			args:          []string{"invoice", "list", "--customer-id", "cus_1"},
			wantErr:       true,
			wantErrSubstr: "store failure",
		},
		{
			name:          "invalid status filter propagated as error",
			service:       &stubInvoiceService{listInvoicesErr: errors.New("invalid invoice status filter")},
			args:          []string{"invoice", "list", "--customer-id", "cus_1", "--status", "pending"},
			wantErr:       true,
			wantErrSubstr: "invalid invoice status filter",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newTestInvoiceCommand(tc.service)
			err := cmd.Run(context.Background(), tc.args, &out)

			if tc.wantErr {
				if err == nil {
					t.Fatal("Run() error = nil, want non-nil")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			got := out.String()
			if tc.wantExact != "" {
				if got != tc.wantExact {
					t.Fatalf("output mismatch:\ngot:\n%s\nwant:\n%s", got, tc.wantExact)
				}
			}
		})
	}
}
