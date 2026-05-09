package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

type stubQuoteService struct {
	createArg app.CreateQuoteCommand
	createRes app.QuoteDTO
	createErr error
	listArg   app.ListQuotesCommand
	listRes   []app.QuoteSummaryDTO
	listErr   error
	getID     string
	getRes    app.QuoteDTO
	getErr    error
	lineArg   app.AddQuoteLineCommand
	lineRes   app.QuoteDTO
	lineErr   error
	sendID    string
	sendRes   app.QuoteDTO
	sendErr   error
	acceptID  string
	acceptRes app.QuoteDTO
	acceptErr error
	rejectID  string
	rejectRes app.QuoteDTO
	rejectErr error
	expireID  string
	expireRes app.QuoteDTO
	expireErr error
	deleteID  string
	deleteErr error
	pdfArg    app.RenderQuotePDFCommand
	pdfRes    app.QuoteRenderedFileDTO
	pdfErr    error
}

func (s *stubQuoteService) Create(ctx context.Context, cmd app.CreateQuoteCommand) (app.QuoteDTO, error) {
	_ = ctx
	s.createArg = cmd
	return s.createRes, s.createErr
}

func (s *stubQuoteService) List(ctx context.Context, cmd app.ListQuotesCommand) ([]app.QuoteSummaryDTO, error) {
	_ = ctx
	s.listArg = cmd
	return s.listRes, s.listErr
}

func (s *stubQuoteService) Get(ctx context.Context, id string) (app.QuoteDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *stubQuoteService) AddLine(ctx context.Context, cmd app.AddQuoteLineCommand) (app.QuoteDTO, error) {
	_ = ctx
	s.lineArg = cmd
	return s.lineRes, s.lineErr
}

func (s *stubQuoteService) Send(ctx context.Context, id string) (app.QuoteDTO, error) {
	_ = ctx
	s.sendID = id
	return s.sendRes, s.sendErr
}

func (s *stubQuoteService) Accept(ctx context.Context, id string) (app.QuoteDTO, error) {
	_ = ctx
	s.acceptID = id
	return s.acceptRes, s.acceptErr
}

func (s *stubQuoteService) Reject(ctx context.Context, id string) (app.QuoteDTO, error) {
	_ = ctx
	s.rejectID = id
	return s.rejectRes, s.rejectErr
}

func (s *stubQuoteService) Expire(ctx context.Context, id string) (app.QuoteDTO, error) {
	_ = ctx
	s.expireID = id
	return s.expireRes, s.expireErr
}

func (s *stubQuoteService) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func (s *stubQuoteService) RenderQuotePDF(ctx context.Context, cmd app.RenderQuotePDFCommand) (app.QuoteRenderedFileDTO, error) {
	_ = ctx
	s.pdfArg = cmd
	return s.pdfRes, s.pdfErr
}

func newTestQuoteCommand(svc QuoteServiceProvider) Command {
	cmd := NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, nil, false)
	return cmd.WithQuoteService(svc)
}

func richQuoteDTOForCLITest(status string) app.QuoteDTO {
	return app.QuoteDTO{ID: "quo_001", CustomerID: "cus_1", Status: status, Currency: "USD", Notes: "Scope", Total: 15000, CanDelete: status != "accepted", CanConvertToInvoice: status == "accepted", CreatedAt: "2026-05-09T20:00:00Z", UpdatedAt: "2026-05-09T20:05:00Z", Lines: []app.QuoteLineDTO{{ID: "qol_001", QuoteID: "quo_001", ServiceAgreementID: "agr_1", Description: "Consulting", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"}}}
}

func TestQuoteCreateCommand(t *testing.T) {
	t.Parallel()
	svc := &stubQuoteService{createRes: richQuoteDTOForCLITest("draft")}
	var out bytes.Buffer
	err := newTestQuoteCommand(svc).Run(context.Background(), []string{"quote", "create", "--customer-id", "cus_1", "--currency", "USD", "--notes", "Scope", "--format", "json"}, &out)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if svc.createArg != (app.CreateQuoteCommand{CustomerID: "cus_1", Currency: "USD", Notes: "Scope"}) {
		t.Fatalf("create arg = %+v", svc.createArg)
	}
	var dto app.QuoteDTO
	if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
		t.Fatalf("json output invalid: %v", err)
	}
	if dto.ID != "quo_001" || dto.Status != "draft" || dto.Total != 15000 || len(dto.Lines) != 1 {
		t.Fatalf("json dto = %+v, want canonical quote", dto)
	}
}

func TestQuoteListShowAndLineCommands(t *testing.T) {
	t.Parallel()
	quote := richQuoteDTOForCLITest("draft")
	tests := []struct {
		name     string
		svc      *stubQuoteService
		args     []string
		wantText string
		check    func(*testing.T, *stubQuoteService, string)
	}{
		{name: "list text", svc: &stubQuoteService{listRes: []app.QuoteSummaryDTO{{ID: "quo_001", CustomerID: "cus_1", Status: "draft", Currency: "USD", Total: 15000}}}, args: []string{"quote", "list", "--customer-id", "cus_1", "--status", "draft"}, wantText: "Quotes (1)", check: func(t *testing.T, svc *stubQuoteService, out string) {
			if svc.listArg != (app.ListQuotesCommand{CustomerID: "cus_1", Status: "draft"}) {
				t.Fatalf("list arg = %+v", svc.listArg)
			}
		}},
		{name: "show toon", svc: &stubQuoteService{getRes: quote}, args: []string{"quote", "show", "--id", "quo_001", "--format", "toon"}, wantText: "can_convert_to_invoice", check: func(t *testing.T, svc *stubQuoteService, out string) {
			if svc.getID != "quo_001" {
				t.Fatalf("get id = %q", svc.getID)
			}
		}},
		{name: "add line json", svc: &stubQuoteService{lineRes: quote}, args: []string{"quote", "add-line", "quo_001", "--agreement-id", "agr_1", "--description", "Consulting", "--minutes", "90", "--format", "json"}, check: func(t *testing.T, svc *stubQuoteService, out string) {
			want := app.AddQuoteLineCommand{QuoteID: "quo_001", ServiceAgreementID: "agr_1", Description: "Consulting", QuantityMin: 90}
			if svc.lineArg != want {
				t.Fatalf("line arg = %+v, want %+v", svc.lineArg, want)
			}
			var dto app.QuoteDTO
			if err := json.Unmarshal([]byte(out), &dto); err != nil || dto.Lines[0].LineTotalAmount != 15000 {
				t.Fatalf("json output = %q, dto=%+v err=%v", out, dto, err)
			}
		}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := newTestQuoteCommand(tc.svc).Run(context.Background(), tc.args, &out)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantText != "" && !strings.Contains(out.String(), tc.wantText) {
				t.Fatalf("output = %q, want contains %q", out.String(), tc.wantText)
			}
			tc.check(t, tc.svc, out.String())
		})
	}
}

func TestQuoteLifecycleAndDeleteCommands(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		svc       *stubQuoteService
		args      []string
		wantID    func(*stubQuoteService) string
		wantText  string
		wantError string
	}{
		{name: "send", svc: &stubQuoteService{sendRes: richQuoteDTOForCLITest("sent")}, args: []string{"quote", "send", "--id", "quo_001"}, wantID: func(s *stubQuoteService) string { return s.sendID }, wantText: "sent"},
		{name: "accept", svc: &stubQuoteService{acceptRes: richQuoteDTOForCLITest("accepted")}, args: []string{"quote", "accept", "--id", "quo_001"}, wantID: func(s *stubQuoteService) string { return s.acceptID }, wantText: "accepted"},
		{name: "reject", svc: &stubQuoteService{rejectRes: richQuoteDTOForCLITest("rejected")}, args: []string{"quote", "reject", "--id", "quo_001"}, wantID: func(s *stubQuoteService) string { return s.rejectID }, wantText: "rejected"},
		{name: "expire", svc: &stubQuoteService{expireRes: richQuoteDTOForCLITest("expired")}, args: []string{"quote", "expire", "--id", "quo_001"}, wantID: func(s *stubQuoteService) string { return s.expireID }, wantText: "expired"},
		{name: "delete", svc: &stubQuoteService{}, args: []string{"quote", "delete", "--id", "quo_001"}, wantID: func(s *stubQuoteService) string { return s.deleteID }, wantText: "Quote deleted"},
		{name: "delete service error", svc: &stubQuoteService{deleteErr: errors.New("accepted quotes cannot be deleted")}, args: []string{"quote", "delete", "--id", "quo_001"}, wantError: "accepted quotes cannot be deleted"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := newTestQuoteCommand(tc.svc).Run(context.Background(), tc.args, &out)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("Run() error = %v, want %q", err, tc.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantID != nil && tc.wantID(tc.svc) != "quo_001" {
				t.Fatalf("command id = %q", tc.wantID(tc.svc))
			}
			if !strings.Contains(out.String(), tc.wantText) {
				t.Fatalf("output = %q, want contains %q", out.String(), tc.wantText)
			}
		})
	}
}

func TestQuotePDFCommandFormatsSharePayload(t *testing.T) {
	t.Parallel()

	result := app.QuoteRenderedFileDTO{QuoteID: "quo_001", Filename: "quo_001.pdf", Path: "/tmp/quo_001.pdf", MimeType: "application/pdf", SizeBytes: 1234}
	tests := []struct {
		name      string
		formatArg string
		wantText  string
		check     func(*testing.T, string)
	}{
		{name: "text", wantText: "Quote PDF Exported", check: func(t *testing.T, out string) {
			if !strings.Contains(out, "Quote") || !strings.Contains(out, "/tmp/quo_001.pdf") || !strings.Contains(out, "1234 bytes") {
				t.Fatalf("text output = %q, want quote pdf metadata", out)
			}
		}},
		{name: "json", formatArg: "json", check: func(t *testing.T, out string) {
			var dto app.QuoteRenderedFileDTO
			if err := json.Unmarshal([]byte(out), &dto); err != nil {
				t.Fatalf("json output invalid: %v", err)
			}
			if dto != result {
				t.Fatalf("json dto = %+v, want %+v", dto, result)
			}
		}},
		{name: "toon", formatArg: "toon", wantText: "quote_id", check: func(t *testing.T, out string) {
			if !strings.Contains(out, "quote_id") || !strings.Contains(out, "size_bytes") || !strings.Contains(out, "application/pdf") {
				t.Fatalf("toon output = %q, want canonical quote pdf fields", out)
			}
		}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := &stubQuoteService{pdfRes: result}
			args := []string{"quote", "pdf", "quo_001", "--out", "/tmp/quo_001.pdf"}
			if tc.formatArg != "" {
				args = append(args, "--format", tc.formatArg)
			}
			var out bytes.Buffer
			err := newTestQuoteCommand(svc).Run(context.Background(), args, &out)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if svc.pdfArg != (app.RenderQuotePDFCommand{QuoteID: "quo_001", OutputPath: "/tmp/quo_001.pdf"}) {
				t.Fatalf("pdf arg = %+v", svc.pdfArg)
			}
			if tc.wantText != "" && !strings.Contains(out.String(), tc.wantText) {
				t.Fatalf("output = %q, want contains %q", out.String(), tc.wantText)
			}
			tc.check(t, out.String())
		})
	}
}

func TestQuoteCommandValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"quote", "create", "--currency", "USD"}, "--customer-id is required"},
		{[]string{"quote", "add-line", "quo_001", "--agreement-id", "agr_1", "--description", "Consulting", "--minutes", "0"}, "--minutes must be greater than 0"},
		{[]string{"quote", "show"}, "--id is required"},
		{[]string{"quote", "pdf"}, "quote id is required"},
		{[]string{"quote", "pdf", "quo_001"}, "--out is required"},
		{[]string{"quote", "wat"}, "unknown command"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			t.Parallel()
			err := newTestQuoteCommand(&stubQuoteService{}).Run(context.Background(), tc.args, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Run() error = %v, want %q", err, tc.want)
			}
		})
	}
}
