package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	inspectID       string
	inspectRes      app.InvoiceDTO
	inspectErr      error
	updateMetaArg   *app.UpdateInvoiceMetadataCommand
	updateMetaRes   app.InvoiceDTO
	updateMetaErr   error
	listInvoicesRes []app.InvoiceSummaryDTO
	listInvoicesErr error
	pdfArg          *app.RenderInvoicePDFCommand
	pdfRes          app.RenderedFileDTO
	pdfErr          error
	addLineArg      *app.AddDraftLineCommand
	addLineRes      app.InvoiceDTO
	addLineErr      error
	removeLineArg   *app.RemoveDraftLineCommand
	removeLineRes   app.InvoiceDTO
	removeLineErr   error
	importArg       *app.ImportIssuedInvoiceCommand
	importRes       app.InvoiceDTO
	importErr       error
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

func (s *stubInvoiceService) InspectInvoice(ctx context.Context, id string) (app.InvoiceDTO, error) {
	_ = ctx
	s.inspectID = id
	return s.inspectRes, s.inspectErr
}

func (s *stubInvoiceService) UpdateInvoiceMetadata(ctx context.Context, cmd app.UpdateInvoiceMetadataCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.updateMetaArg = &cmd
	return s.updateMetaRes, s.updateMetaErr
}

func (s *stubInvoiceService) ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]app.InvoiceSummaryDTO, error) {
	_ = ctx
	return s.listInvoicesRes, s.listInvoicesErr
}

func (s *stubInvoiceService) RenderInvoicePDF(ctx context.Context, cmd app.RenderInvoicePDFCommand) (app.RenderedFileDTO, error) {
	_ = ctx
	s.pdfArg = &cmd
	return s.pdfRes, s.pdfErr
}

func (s *stubInvoiceService) AddDraftLine(ctx context.Context, cmd app.AddDraftLineCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.addLineArg = &cmd
	return s.addLineRes, s.addLineErr
}

func (s *stubInvoiceService) RemoveDraftLine(ctx context.Context, cmd app.RemoveDraftLineCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.removeLineArg = &cmd
	return s.removeLineRes, s.removeLineErr
}

func (s *stubInvoiceService) ImportIssued(ctx context.Context, cmd app.ImportIssuedInvoiceCommand) (app.InvoiceDTO, error) {
	_ = ctx
	s.importArg = &cmd
	return s.importRes, s.importErr
}

func newTestInvoiceCommand(svc InvoiceServiceProvider) Command {
	return NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, nil, svc, false)
}

func richInvoiceDTOForCLITest() app.InvoiceDTO {
	return app.InvoiceDTO{
		ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD",
		InvoiceDate: "2026-05-01T00:00:00Z", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z",
		PaymentTerms: "Net 15", PaymentCommunication: "Use INV-001", ExternalNumber: "EXT-001", ImportSource: "manual-pdf-extract", ImportedAt: "2026-05-01T12:00:00Z",
		IssuedAt: "2026-05-01T09:00:00Z", CreatedAt: "2026-05-01T08:00:00Z", UpdatedAt: "2026-05-01T10:00:00Z",
		Lines:    []app.InvoiceLineDTO{{ID: "inl_001", Description: "Consulting", QuantityMin: 90, UnitRateAmount: 10000, UnitRateCurrency: "USD", LineTotalAmount: 15000, LineTotalCurrency: "USD"}},
		Subtotal: 15000, GrandTotal: 15000,
	}
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
				draftRes: app.InvoiceDTO{ID: "inv_001", CustomerID: "cus_1", Status: "draft", IsDraft: true, PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", Notes: "Net 15"},
			},
			args:        []string{"invoice", "draft", "--customer-id", "cus_1", "--period-start", "2026-04-01", "--period-end", "2026-04-30", "--due-date", "2026-05-15", "--notes", "Net 15"},
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
			if tc.service.draftArg != nil && tc.service.draftArg.PeriodStart != "2026-04-01" {
				t.Fatalf("draft arg = %+v, want metadata flags passed through", tc.service.draftArg)
			}
		})
	}
}

func TestInvoiceLineCommands(t *testing.T) {
	t.Parallel()

	updated := app.InvoiceDTO{ID: "inv_001", CustomerID: "cus_1", Status: "draft", Currency: "USD", IsDraft: true, Lines: []app.InvoiceLineDTO{{ID: "inl_manual", Description: "Setup fee", QuantityMin: 60, UnitRateAmount: 5000, UnitRateCurrency: "USD", LineTotalAmount: 5000, LineTotalCurrency: "USD"}}, Subtotal: 5000, GrandTotal: 5000}
	tests := []struct {
		name        string
		svc         *stubInvoiceService
		args        []string
		wantErr     string
		wantJSON    bool
		wantAdd     *app.AddDraftLineCommand
		wantRemove  *app.RemoveDraftLineCommand
		wantContain string
	}{
		{name: "add json canonical dto", svc: &stubInvoiceService{addLineRes: updated}, args: []string{"invoice", "line", "add", "inv_001", "--description", "Setup fee", "--minutes", "60", "--rate", "5000", "--currency", "USD", "--format", "json"}, wantJSON: true, wantAdd: &app.AddDraftLineCommand{InvoiceID: "inv_001", Description: "Setup fee", QuantityMin: 60, UnitRate: 5000, Currency: "USD"}},
		{name: "remove toon canonical dto", svc: &stubInvoiceService{removeLineRes: updated}, args: []string{"invoice", "line", "remove", "inv_001", "inl_manual", "--format", "toon"}, wantRemove: &app.RemoveDraftLineCommand{InvoiceID: "inv_001", InvoiceLineID: "inl_manual"}, wantContain: "grand_total"},
		{name: "add validation error", svc: &stubInvoiceService{}, args: []string{"invoice", "line", "add", "inv_001", "--minutes", "60", "--rate", "5000", "--currency", "USD"}, wantErr: "--description is required"},
		{name: "remove service error", svc: &stubInvoiceService{removeLineErr: errors.New("invoice is not draft")}, args: []string{"invoice", "line", "remove", "inv_001", "inl_manual"}, wantErr: "invoice is not draft"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := newTestInvoiceCommand(tc.svc).Run(context.Background(), tc.args, &out)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Run() error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantAdd != nil && (tc.svc.addLineArg == nil || *tc.svc.addLineArg != *tc.wantAdd) {
				t.Fatalf("add arg = %+v, want %+v", tc.svc.addLineArg, tc.wantAdd)
			}
			if tc.wantRemove != nil && (tc.svc.removeLineArg == nil || *tc.svc.removeLineArg != *tc.wantRemove) {
				t.Fatalf("remove arg = %+v, want %+v", tc.svc.removeLineArg, tc.wantRemove)
			}
			if tc.wantJSON {
				var dto app.InvoiceDTO
				if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
					t.Fatalf("json output invalid: %v", err)
				}
				if dto.ID != "inv_001" || dto.GrandTotal != 5000 || len(dto.Lines) != 1 || dto.Lines[0].Description != "Setup fee" {
					t.Fatalf("json dto = %+v, want canonical invoice with manual line", dto)
				}
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Fatalf("output = %q, want %q", out.String(), tc.wantContain)
			}
		})
	}
}

func TestInvoiceImportCommandFileStdinAndFormats(t *testing.T) {
	t.Parallel()

	dto := app.InvoiceDTO{ID: "inv_imported", InvoiceNumber: "INV/2026/00001", CustomerID: "cus_1", Status: "issued", Currency: "USD", GrandTotal: 250000, Lines: []app.InvoiceLineDTO{{Description: "Software", AmountMinor: 250000, QuantityDisplay: "160.00"}}}
	payload := `{"schema":"billar.invoice.import/v1","invoice_number":"INV/2026/00001","invoice_date":"2026-02-02","due_date":"2026-02-17","currency":"USD","customer":{"customer_profile_id":"cus_1"},"issuer":{"issuer_profile_id":"iss_1"},"lines":[{"description":"Software","quantity_display":"160.00","unit_price_display":"15.6250","amount_minor":250000}],"totals":{"subtotal_minor":250000,"tax_total_minor":0,"grand_total_minor":250000}}`
	file := filepath.Join(t.TempDir(), "invoice.json")
	if err := os.WriteFile(file, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	t.Run("file text output", func(t *testing.T) {
		svc := &stubInvoiceService{importRes: dto}
		var out bytes.Buffer
		if err := newTestInvoiceCommand(svc).Run(context.Background(), []string{"invoice", "import", "--file", file}, &out); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if svc.importArg == nil || svc.importArg.Payload.InvoiceNumber != "INV/2026/00001" || len(svc.importArg.Payload.Lines) != 1 {
			t.Fatalf("import arg = %+v, want decoded payload", svc.importArg)
		}
		if !strings.Contains(out.String(), "Invoice imported: inv_imported") || !strings.Contains(out.String(), "INV/2026/00001") || !strings.Contains(out.String(), "250000") {
			t.Fatalf("output = %q, want import summary", out.String())
		}
	})

	t.Run("stdin json output", func(t *testing.T) {
		svc := &stubInvoiceService{importRes: dto}
		cmd := newTestInvoiceCommand(svc)
		cmd.stdin = strings.NewReader(payload)
		var out bytes.Buffer
		if err := cmd.Run(context.Background(), []string{"invoice", "import", "--stdin", "--format", "json"}, &out); err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		var got app.InvoiceDTO
		if err := json.Unmarshal(out.Bytes(), &got); err != nil {
			t.Fatalf("json output: %v", err)
		}
		if got.InvoiceNumber != dto.InvoiceNumber || got.GrandTotal != dto.GrandTotal {
			t.Fatalf("dto = %+v, want canonical import DTO", got)
		}
	})
}

func TestInvoicePDFCommandWritesConfirmationFormats(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outPath := filepath.Join(dir, "inv_001.pdf")
	want := app.RenderedFileDTO{InvoiceID: "inv_001", Filename: "inv_001.pdf", Path: outPath, MimeType: "application/pdf", SizeBytes: 9}
	formats := []string{"text", "json", "toon"}
	for _, format := range formats {
		format := format
		t.Run(format, func(t *testing.T) {
			svc := &stubInvoiceService{pdfRes: want}
			var out bytes.Buffer
			cmd := newTestInvoiceCommand(svc)
			err := cmd.Run(context.Background(), []string{"invoice", "pdf", "inv_001", "--out", outPath, "--format", format}, &out)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if svc.pdfArg == nil || svc.pdfArg.InvoiceID != "inv_001" || svc.pdfArg.OutputPath != outPath {
				t.Fatalf("pdf arg = %+v", svc.pdfArg)
			}
			got := out.String()
			for _, required := range []string{"inv_001", "inv_001.pdf", outPath, "application/pdf", "9"} {
				if !strings.Contains(got, required) {
					t.Fatalf("%s output = %q, want invoice PDF metadata value %q", format, got, required)
				}
			}
			for _, forbidden := range []string{"quote_id", "QUOTE PROPOSAL", "proposal"} {
				if strings.Contains(got, forbidden) {
					t.Fatalf("%s output = %q, must preserve invoice vocabulary and not contain %q", format, got, forbidden)
				}
			}
			switch format {
			case "text":
				if !strings.Contains(got, "Invoice PDF Exported") {
					t.Fatalf("text output = %q, want invoice-specific confirmation", got)
				}
			case "json":
				var dto app.RenderedFileDTO
				if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
					t.Fatalf("json output invalid: %v", err)
				}
				if dto != want {
					t.Fatalf("json dto = %+v, want unchanged RenderedFileDTO %+v", dto, want)
				}
			case "toon":
				for _, field := range []string{"invoice_id", "filename", "path", "mime_type", "size_bytes"} {
					if !strings.Contains(got, field) {
						t.Fatalf("toon output = %q, want RenderedFileDTO field %q", got, field)
					}
				}
			}
		})
	}
}

func TestInvoicePDFCommandAllowsDefaultExportDirWhenOutOmitted(t *testing.T) {
	t.Parallel()

	svc := &stubInvoiceService{pdfRes: app.RenderedFileDTO{InvoiceID: "inv_001", Filename: "invoice-INV-001.pdf", Path: filepath.Join(t.TempDir(), "invoice-INV-001.pdf"), MimeType: "application/pdf", SizeBytes: 9}}
	cmd := newTestInvoiceCommand(svc)
	cmd.exportDir = t.TempDir()
	var out bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"invoice", "pdf", "inv_001"}, &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if svc.pdfArg == nil || svc.pdfArg.InvoiceID != "inv_001" || svc.pdfArg.OutputPath != "" {
		t.Fatalf("pdf arg = %+v, want invoice id and default output path", svc.pdfArg)
	}
	if !strings.Contains(out.String(), "invoice-INV-001.pdf") {
		t.Fatalf("output = %q, want default filename", out.String())
	}
}

func TestInvoiceInspectCommandFormatsAndErrors(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"text", "json", "toon"} {
		format := format
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			svc := &stubInvoiceService{inspectRes: richInvoiceDTOForCLITest()}
			var out bytes.Buffer
			err := newTestInvoiceCommand(svc).Run(context.Background(), []string{"invoice", "inspect", "--id", "inv_001", "--format", format}, &out)
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if svc.inspectID != "inv_001" {
				t.Fatalf("inspectID = %q, want inv_001", svc.inspectID)
			}
			got := out.String()
			for _, want := range []string{"INV-001", "invoice_date", "payment_terms", "import_source", "grand_total"} {
				if !strings.Contains(got, want) {
					t.Fatalf("%s output missing %q:\n%s", format, want, got)
				}
			}
			if format == "json" {
				var dto app.InvoiceDTO
				if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
					t.Fatalf("json output invalid: %v", err)
				}
				if dto.InvoiceDate != "2026-05-01T00:00:00Z" || dto.ImportSource != "manual-pdf-extract" || len(dto.Lines) != 1 {
					t.Fatalf("json dto = %+v, want canonical invoice metadata and lines", dto)
				}
			}
		})
	}

	t.Run("not found exits non-zero without payload", func(t *testing.T) {
		t.Parallel()
		svc := &stubInvoiceService{inspectErr: app.ErrInvoiceNotFound}
		var out bytes.Buffer
		err := newTestInvoiceCommand(svc).Run(context.Background(), []string{"invoice", "inspect", "--id", "inv_missing", "--format", "json"}, &out)
		if err == nil || !strings.Contains(err.Error(), "invoice not found") {
			t.Fatalf("Run() error = %v, want invoice not found", err)
		}
		if out.Len() != 0 {
			t.Fatalf("output = %q, want no partial payload", out.String())
		}
	})
}

func TestInvoiceUpdateMetadataCommand(t *testing.T) {
	t.Parallel()

	t.Run("success formats canonical dto and preserves imported identity", func(t *testing.T) {
		t.Parallel()
		svc := &stubInvoiceService{updateMetaRes: richInvoiceDTOForCLITest()}
		var out bytes.Buffer
		err := newTestInvoiceCommand(svc).Run(context.Background(), []string{"invoice", "update-metadata", "--id", "inv_001", "--invoice-date", "2026-05-01", "--payment-terms", "Net 15", "--payment-communication", "Use INV-001", "--format", "json"}, &out)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		want := app.UpdateInvoiceMetadataCommand{InvoiceID: "inv_001", InvoiceDate: "2026-05-01", PaymentTerms: "Net 15", PaymentCommunication: "Use INV-001"}
		if svc.updateMetaArg == nil || *svc.updateMetaArg != want {
			t.Fatalf("update metadata arg = %+v, want %+v", svc.updateMetaArg, want)
		}
		var dto app.InvoiceDTO
		if err := json.Unmarshal(out.Bytes(), &dto); err != nil {
			t.Fatalf("json output invalid: %v", err)
		}
		if dto.ImportSource != "manual-pdf-extract" || dto.ExternalNumber != "EXT-001" || dto.GrandTotal != 15000 || len(dto.Lines) != 1 {
			t.Fatalf("dto = %+v, want imported identity and financial fields preserved", dto)
		}
	})

	t.Run("date validation error writes nothing", func(t *testing.T) {
		t.Parallel()
		svc := &stubInvoiceService{updateMetaErr: errors.New("invoice_date must be YYYY-MM-DD or RFC3339")}
		var out bytes.Buffer
		err := newTestInvoiceCommand(svc).Run(context.Background(), []string{"invoice", "update-metadata", "--id", "inv_001", "--invoice-date", "not-a-date"}, &out)
		if err == nil || !strings.Contains(err.Error(), "invoice_date") {
			t.Fatalf("Run() error = %v, want invoice_date validation", err)
		}
		if out.Len() != 0 {
			t.Fatalf("output = %q, want empty on error", out.String())
		}
	})

	t.Run("toon output uses same canonical dto", func(t *testing.T) {
		t.Parallel()
		svc := &stubInvoiceService{updateMetaRes: richInvoiceDTOForCLITest()}
		var out bytes.Buffer
		err := newTestInvoiceCommand(svc).Run(context.Background(), []string{"invoice", "update-metadata", "--id", "inv_001", "--due-date", "2026-05-15", "--format", "toon"}, &out)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !strings.Contains(out.String(), "payment_communication") || !strings.Contains(out.String(), "import_source") || !strings.Contains(out.String(), "grand_total") {
			t.Fatalf("toon output = %q, want canonical invoice fields", out.String())
		}
	})
}

func TestInvoicePDFCommandRejectsMissingArgumentsAndPropagatesErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		svc  *stubInvoiceService
		want string
	}{
		{"missing invoice id", []string{"invoice", "pdf", "--out", "x.pdf"}, &stubInvoiceService{}, "invoice id is required"},
		{"missing export dir and out", []string{"invoice", "pdf", "inv_001"}, &stubInvoiceService{}, "BILLAR_EXPORT_DIR"},
		{"write error", []string{"invoice", "pdf", "inv_001", "--out", filepath.Join(t.TempDir(), "missing", "x.pdf")}, &stubInvoiceService{pdfErr: errors.New("write file: no such file or directory")}, "write file"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := newTestInvoiceCommand(tc.svc).Run(context.Background(), tc.args, &out)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Run() error = %v, want %q", err, tc.want)
			}
			if tc.name == "missing export dir and out" && !strings.Contains(err.Error(), "--out") {
				t.Fatalf("Run() error = %v, want --out guidance", err)
			}
			if out.Len() != 0 {
				t.Fatalf("output = %q, want empty on error", out.String())
			}
		})
	}
	_ = os.ErrNotExist
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
		"Period: 2026-04-01T00:00:00Z — 2026-04-30T00:00:00Z\n" +
		"Due Date: 2026-05-15T00:00:00Z\n" +
		"Notes: Net 15\n" +
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
					ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", Notes: "Net 15",
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
		"Number           Status       Period                       Due Date              Currency   Total\n" +
		"INV-001          issued       2026-04-01T00:00:00Z–2026-04-30T00:00:00Z 2026-05-15T00:00:00Z  USD        15000\n" +
		"—                draft        –                            –                     USD        3000\n"

	wantFilterExact := "Invoices\n" +
		"────────\n" +
		"Customer: cus_1\n" +
		"Status: draft\n" +
		"Count: 1\n" +
		"\n" +
		"Number           Status       Period                       Due Date              Currency   Total\n" +
		"—                draft        –                            –                     USD        3000\n"

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
					{ID: "inv_001", InvoiceNumber: "INV-001", CustomerID: "cus_1", Status: "issued", Currency: "USD", PeriodStart: "2026-04-01T00:00:00Z", PeriodEnd: "2026-04-30T00:00:00Z", DueDate: "2026-05-15T00:00:00Z", GrandTotal: 15000},
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
