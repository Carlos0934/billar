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
	draftArg   *app.CreateDraftFromUnbilledCommand
	draftRes   app.InvoiceDTO
	draftErr   error
	issueArg   *app.IssueInvoiceCommand
	issueRes   app.InvoiceDTO
	issueErr   error
	discardID  string
	discardRes app.DiscardResult
	discardErr error
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
