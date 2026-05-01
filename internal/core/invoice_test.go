package core

import (
	"strings"
	"testing"
	"time"
)

func TestInvoiceStatusIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status InvoiceStatus
		want   bool
	}{
		{status: InvoiceStatusDraft, want: true},
		{status: InvoiceStatusIssued, want: true},
		{status: InvoiceStatusDiscarded, want: true},
		{status: InvoiceStatus("voided"), want: false},
		{status: InvoiceStatus("pending"), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			if got := tt.status.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewInvoiceLine(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}

	line, err := NewInvoiceLine(InvoiceLineParams{
		InvoiceID:          "inv_123",
		ServiceAgreementID: "sa_123",
		TimeEntryID:        "te_123",
		UnitRate:           rate,
	})
	if err != nil {
		t.Fatalf("NewInvoiceLine() error = %v", err)
	}
	if !strings.HasPrefix(line.ID, "inl_") {
		t.Fatalf("ID = %q, want inl_ prefix", line.ID)
	}
	if line.InvoiceID != "inv_123" || line.ServiceAgreementID != "sa_123" || line.TimeEntryID != "te_123" {
		t.Fatalf("NewInvoiceLine() = %#v, want all IDs preserved", line)
	}

	_, err = NewInvoiceLine(InvoiceLineParams{InvoiceID: "", ServiceAgreementID: "sa_123", TimeEntryID: "te_123", UnitRate: rate})
	if err == nil {
		t.Fatal("NewInvoiceLine() error = nil, want blank invoice id rejected")
	}
	_, err = NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_123", ServiceAgreementID: "", TimeEntryID: "te_123", UnitRate: rate})
	if err == nil {
		t.Fatal("NewInvoiceLine() error = nil, want blank service agreement id rejected")
	}
	_, err = NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_123", ServiceAgreementID: "sa_123", TimeEntryID: "", UnitRate: rate})
	if err == nil {
		t.Fatal("NewInvoiceLine() error = nil, want blank time entry id rejected")
	}
}

func TestInvoiceLineLineTotal(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	line := InvoiceLine{UnitRate: rate}

	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{Hours: hours}

	total := line.LineTotal(entry)
	if total.Amount != 15000 {
		t.Fatalf("LineTotal() amount = %d, want 15000", total.Amount)
	}
	if total.Currency != "USD" {
		t.Fatalf("LineTotal() currency = %q, want USD", total.Currency)
	}
}

func TestNewManualInvoiceLineAndSnapshotTotals(t *testing.T) {
	t.Parallel()

	usdRate, _ := NewMoney(5000, "USD")
	eurRate, _ := NewMoney(5000, "EUR")
	tests := []struct {
		name          string
		description   string
		quantityMin   int64
		rate          Money
		invoiceCurr   string
		wantErrSubstr string
	}{
		{name: "valid manual line", description: "Setup fee", quantityMin: 60, rate: usdRate, invoiceCurr: "USD"},
		{name: "blank description", description: "  ", quantityMin: 60, rate: usdRate, invoiceCurr: "USD", wantErrSubstr: "description"},
		{name: "zero quantity", description: "Setup fee", quantityMin: 0, rate: usdRate, invoiceCurr: "USD", wantErrSubstr: "quantity"},
		{name: "zero rate", description: "Setup fee", quantityMin: 60, rate: Money{Currency: "USD"}, invoiceCurr: "USD", wantErrSubstr: "unit rate"},
		{name: "currency mismatch", description: "Setup fee", quantityMin: 60, rate: eurRate, invoiceCurr: "USD", wantErrSubstr: "currency"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			line, err := NewManualInvoiceLine("inv_123", "Setup fee", tt.description, tt.quantityMin, tt.rate, tt.invoiceCurr)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("NewManualInvoiceLine() error = %v, want %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewManualInvoiceLine() error = %v", err)
			}
			if line.TimeEntryID != "" || line.Description != "Setup fee" || line.QuantityMin != 60 {
				t.Fatalf("line snapshot = %+v, want manual Setup fee 60min", line)
			}
			if got := line.LineTotal(); got.Amount != 5000 || got.Currency != "USD" {
				t.Fatalf("LineTotal() = %+v, want 5000 USD", got)
			}
		})
	}
}

func TestInvoiceAddAndRemoveLineInvariants(t *testing.T) {
	t.Parallel()

	rate, _ := NewMoney(6000, "USD")
	base, _ := NewManualInvoiceLine("inv_seed", "sa_123", "Base", 60, rate, "USD")
	extra, _ := NewManualInvoiceLine("inv_seed", "sa_123", "Extra", 30, rate, "USD")
	invoice, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{base}})
	if err != nil {
		t.Fatalf("NewInvoice() error = %v", err)
	}
	if err := invoice.AddLine(extra); err != nil {
		t.Fatalf("AddLine() error = %v", err)
	}
	if len(invoice.Lines) != 2 || invoice.Lines[1].InvoiceID != invoice.ID {
		t.Fatalf("Lines after AddLine = %+v, want appended with invoice id %q", invoice.Lines, invoice.ID)
	}
	if err := invoice.RemoveLine(extra.ID); err != nil {
		t.Fatalf("RemoveLine(extra) error = %v", err)
	}
	if len(invoice.Lines) != 1 || invoice.Lines[0].ID != base.ID {
		t.Fatalf("Lines after RemoveLine = %+v, want only base", invoice.Lines)
	}
	if err := invoice.RemoveLine(base.ID); err == nil || !strings.Contains(err.Error(), "last") {
		t.Fatalf("RemoveLine(last) error = %v, want final-line rejection", err)
	}
	if err := invoice.AddLine(InvoiceLine{ID: "inl_eur", UnitRate: Money{Amount: 1, Currency: "EUR"}, QuantityMin: 1, Description: "bad"}); err == nil || !strings.Contains(err.Error(), "currency") {
		t.Fatalf("AddLine(currency mismatch) error = %v, want currency rejection", err)
	}
}

func TestNewInvoice(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{ID: "te_123", Hours: hours}
	line, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: entry.ID, UnitRate: rate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(): %v", err)
	}

	invoice, err := NewInvoice(InvoiceParams{
		CustomerID: "cus_123",
		Status:     InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []InvoiceLine{line},
		CreatedAt:  time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewInvoice() error = %v", err)
	}
	if !strings.HasPrefix(invoice.ID, "inv_") {
		t.Fatalf("ID = %q, want inv_ prefix", invoice.ID)
	}
	if !invoice.IsDraft() {
		t.Fatal("IsDraft() = false, want true")
	}
	if len(invoice.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1", len(invoice.Lines))
	}
	if invoice.Lines[0].InvoiceID != invoice.ID {
		t.Fatalf("line InvoiceID = %q, want %q", invoice.Lines[0].InvoiceID, invoice.ID)
	}
	total := invoice.Total([]TimeEntry{entry})
	if total.Amount != 15000 {
		t.Fatalf("Total() amount = %d, want 15000", total.Amount)
	}

	_, err = NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: nil})
	if err == nil {
		t.Fatal("NewInvoice() error = nil, want zero lines rejected")
	}

	otherRate, err := NewMoney(10000, "EUR")
	if err != nil {
		t.Fatalf("NewMoney(otherRate): %v", err)
	}
	badLine, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_456", TimeEntryID: "te_456", UnitRate: otherRate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(other): %v", err)
	}
	_, err = NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{badLine}})
	if err == nil {
		t.Fatal("NewInvoice() error = nil, want currency mismatch rejected")
	}
}

func TestNewInvoiceMetadataValidation(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	line, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: "te_123", UnitRate: rate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(): %v", err)
	}
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		params  InvoiceParams
		wantErr string
	}{
		{
			name:   "accepts explicit metadata",
			params: InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}, PeriodStart: periodStart, PeriodEnd: periodEnd, DueDate: periodEnd.AddDate(0, 0, 15), Notes: "Net 15"},
		},
		{
			name:   "accepts unset period and due date",
			params: InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}, Notes: ""},
		},
		{
			name:    "rejects period end before start",
			params:  InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}, PeriodStart: periodEnd, PeriodEnd: periodStart},
			wantErr: "period_end must be on or after period_start",
		},
		{
			name:    "rejects due date before period end",
			params:  InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}, PeriodStart: periodStart, PeriodEnd: periodEnd, DueDate: periodStart},
			wantErr: "due_date must be on or after period_end",
		},
		{
			name:    "rejects due date before period start when end unset",
			params:  InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}, PeriodStart: periodStart, DueDate: periodStart.AddDate(0, 0, -1)},
			wantErr: "due_date must be on or after period_start",
		},
		{
			name:    "rejects notes over maximum length",
			params:  InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}, Notes: strings.Repeat("x", 4001)},
			wantErr: "invoice notes must be 4000 characters or fewer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			invoice, err := NewInvoice(tt.params)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("NewInvoice() error = %v, want substring %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewInvoice() error = %v", err)
			}
			if !invoice.PeriodStart.Equal(tt.params.PeriodStart) || !invoice.PeriodEnd.Equal(tt.params.PeriodEnd) || !invoice.DueDate.Equal(tt.params.DueDate) || invoice.Notes != tt.params.Notes {
				t.Fatalf("metadata = (%s,%s,%s,%q), want (%s,%s,%s,%q)", invoice.PeriodStart, invoice.PeriodEnd, invoice.DueDate, invoice.Notes, tt.params.PeriodStart, tt.params.PeriodEnd, tt.params.DueDate, tt.params.Notes)
			}
		})
	}
}

func TestInvoiceUpdateMetadata(t *testing.T) {
	t.Parallel()

	original := testInvoiceForMetadataUpdate(t, InvoiceStatusIssued)
	original.IssuedAt = time.Date(2026, 4, 10, 9, 30, 0, 0, time.UTC)
	original.InvoiceNumber = "INV-2026-0001"
	original.ImportSource = "manual-pdf-extract"
	original.ImportedAt = time.Date(2026, 4, 26, 13, 25, 34, 0, time.UTC)
	original.ExternalNumber = "EXT-OLD"
	original.CreatedAt = time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)
	original.UpdatedAt = original.CreatedAt
	original.Lines = []InvoiceLine{{ID: "inl_keep", InvoiceID: original.ID, Description: "Development", QuantityMin: 60, UnitRate: Money{Amount: 10000, Currency: "USD"}}}

	updated := original
	newInvoiceDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	newPeriodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	newPeriodEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	newDueDate := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)

	if err := updated.UpdateMetadata(InvoiceMetadataPatch{
		InvoiceDate:          newInvoiceDate,
		PeriodStart:          newPeriodStart,
		PeriodEnd:            newPeriodEnd,
		DueDate:              newDueDate,
		PaymentTerms:         "Net 15",
		PaymentCommunication: "Use invoice INV-2026-0001",
		Notes:                "Updated notes",
		ExternalNumber:       "EXT-NEW",
	}); err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	if !updated.InvoiceDate.Equal(newInvoiceDate) || !updated.PeriodStart.Equal(newPeriodStart) || !updated.PeriodEnd.Equal(newPeriodEnd) || !updated.DueDate.Equal(newDueDate) {
		t.Fatalf("updated dates = (%s,%s,%s,%s), want (%s,%s,%s,%s)", updated.InvoiceDate, updated.PeriodStart, updated.PeriodEnd, updated.DueDate, newInvoiceDate, newPeriodStart, newPeriodEnd, newDueDate)
	}
	if updated.PaymentTerms != "Net 15" || updated.PaymentCommunication != "Use invoice INV-2026-0001" || updated.Notes != "Updated notes" || updated.ExternalNumber != "EXT-NEW" {
		t.Fatalf("updated strings = (%q,%q,%q,%q), want metadata strings", updated.PaymentTerms, updated.PaymentCommunication, updated.Notes, updated.ExternalNumber)
	}
	if updated.InvoiceNumber != original.InvoiceNumber || !updated.IssuedAt.Equal(original.IssuedAt) || !updated.CreatedAt.Equal(original.CreatedAt) || updated.ImportSource != original.ImportSource || !updated.ImportedAt.Equal(original.ImportedAt) {
		t.Fatalf("immutable identity changed: before=%+v after=%+v", original, updated)
	}
	if len(updated.Lines) != len(original.Lines) || updated.Lines[0].ID != original.Lines[0].ID || updated.Total(nil).Amount != original.Total(nil).Amount {
		t.Fatalf("financial data changed: before lines=%+v total=%+v after lines=%+v total=%+v", original.Lines, original.Total(nil), updated.Lines, updated.Total(nil))
	}
	if !updated.UpdatedAt.After(original.UpdatedAt) {
		t.Fatalf("UpdatedAt = %s, want after %s", updated.UpdatedAt, original.UpdatedAt)
	}
}

func TestInvoiceUpdateMetadataValidation(t *testing.T) {
	t.Parallel()

	validPeriodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	validPeriodEnd := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	validDueDate := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		status     InvoiceStatus
		patch      InvoiceMetadataPatch
		wantErr    string
		wantDueDay int
	}{
		{
			name:       "draft accepts metadata patch",
			status:     InvoiceStatusDraft,
			patch:      InvoiceMetadataPatch{PeriodStart: validPeriodStart, PeriodEnd: validPeriodEnd, DueDate: validDueDate, PaymentTerms: "Net 15"},
			wantDueDay: 15,
		},
		{
			name:    "discarded invoices are rejected",
			status:  InvoiceStatusDiscarded,
			patch:   InvoiceMetadataPatch{PaymentTerms: "Net 15"},
			wantErr: "discarded",
		},
		{
			name:    "due date before period end is rejected",
			status:  InvoiceStatusIssued,
			patch:   InvoiceMetadataPatch{PeriodStart: validPeriodStart, PeriodEnd: validPeriodEnd, DueDate: validPeriodStart},
			wantErr: "due_date must be on or after period_end",
		},
		{
			name:    "period end before period start is rejected",
			status:  InvoiceStatusIssued,
			patch:   InvoiceMetadataPatch{PeriodStart: validPeriodEnd, PeriodEnd: validPeriodStart},
			wantErr: "period_end must be on or after period_start",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			invoice := testInvoiceForMetadataUpdate(t, tt.status)
			before := invoice
			err := invoice.UpdateMetadata(tt.patch)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("UpdateMetadata() error = %v, want substring %q", err, tt.wantErr)
				}
				if !invoice.DueDate.Equal(before.DueDate) || invoice.PaymentTerms != before.PaymentTerms || !invoice.UpdatedAt.Equal(before.UpdatedAt) {
					t.Fatalf("metadata changed after rejected patch: before=%+v after=%+v", before, invoice)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateMetadata() error = %v", err)
			}
			if invoice.DueDate.Day() != tt.wantDueDay || invoice.PaymentTerms != tt.patch.PaymentTerms {
				t.Fatalf("updated invoice = %+v, want due day %d and payment terms %q", invoice, tt.wantDueDay, tt.patch.PaymentTerms)
			}
		})
	}
}

func testInvoiceForMetadataUpdate(t *testing.T, status InvoiceStatus) Invoice {
	t.Helper()
	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	line, err := NewManualInvoiceLine("inv_seed", "sa_123", "Development", 60, rate, "USD")
	if err != nil {
		t.Fatalf("NewManualInvoiceLine(): %v", err)
	}
	invoice, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: status, Currency: "USD", Lines: []InvoiceLine{line}, CreatedAt: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("NewInvoice(): %v", err)
	}
	return invoice
}

func TestInvoiceIssue_HappyPath(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{ID: "te_123", Hours: hours}
	line, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: entry.ID, UnitRate: rate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(): %v", err)
	}
	invoice, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}})
	if err != nil {
		t.Fatalf("NewInvoice(): %v", err)
	}

	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if invoice.Status != InvoiceStatusIssued {
		t.Fatalf("Status = %q, want issued", invoice.Status)
	}
	if invoice.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", invoice.InvoiceNumber)
	}
	if !invoice.IssuedAt.Equal(issuedAt) {
		t.Fatalf("IssuedAt = %s, want %s", invoice.IssuedAt, issuedAt)
	}
	if !invoice.UpdatedAt.Equal(issuedAt) {
		t.Fatalf("UpdatedAt = %s, want %s", invoice.UpdatedAt, issuedAt)
	}

	if err := invoice.Issue("INV-2026-0002", issuedAt.Add(time.Hour)); err == nil {
		t.Fatal("Issue() error = nil, want reissue rejected")
	}
}

func TestInvoiceIssue_RejectsNonDraft(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusIssued}
	if err := invoice.Issue("INV-1", time.Now().UTC()); err == nil {
		t.Fatal("Issue() error = nil, want non-draft rejected")
	}
}

func TestInvoiceIssue_RejectsInvalidInputs(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusDraft}
	if err := invoice.Issue("", time.Now().UTC()); err == nil {
		t.Fatal("Issue() error = nil, want blank number rejected")
	}
	if err := invoice.Issue("INV-1", time.Time{}); err == nil {
		t.Fatal("Issue() error = nil, want zero issued time rejected")
	}
}

func TestInvoiceIsIssued(t *testing.T) {
	t.Parallel()

	issued := Invoice{Status: InvoiceStatusIssued}
	if !issued.IsIssued() {
		t.Fatal("IsIssued() = false, want true")
	}
	if (Invoice{Status: InvoiceStatusDraft}).IsIssued() {
		t.Fatal("IsIssued() = true for draft invoice")
	}
}

func TestInvoiceNewInvoiceLineErrors(t *testing.T) {
	t.Parallel()

	_, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	_, err = NewInvoice(InvoiceParams{CustomerID: "", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{{}}})
	if err == nil {
		t.Fatal("NewInvoice() error = nil, want blank customer id rejected")
	}
}

func TestNewInvoiceRejectsInvalidStatus(t *testing.T) {
	t.Parallel()

	if _, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatus("pending"), Currency: "USD", Lines: []InvoiceLine{{InvoiceID: "inv_x", UnitRate: Money{Amount: 1, Currency: "USD"}}}}); err == nil {
		t.Fatal("NewInvoice() error = nil, want invalid status rejected")
	}
}

func TestInvoiceDiscard_IssuedToDiscarded(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusIssued}
	discardedAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)

	if err := invoice.Discard(discardedAt); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if invoice.Status != InvoiceStatusDiscarded {
		t.Fatalf("Status = %q, want discarded", invoice.Status)
	}
	if !invoice.DiscardedAt.Equal(discardedAt) {
		t.Fatalf("DiscardedAt = %v, want %v", invoice.DiscardedAt, discardedAt)
	}
}

func TestInvoiceDiscard_RejectsDraft(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusDraft}
	if err := invoice.Discard(time.Now().UTC()); err == nil {
		t.Fatal("Discard() error = nil, want draft rejection")
	}
}

func TestInvoiceDiscard_RejectsAlreadyDiscarded(t *testing.T) {
	t.Parallel()

	invoice := Invoice{Status: InvoiceStatusDiscarded}
	if err := invoice.Discard(time.Now().UTC()); err == nil {
		t.Fatal("Discard() error = nil, want already-discarded rejection")
	}
}

func TestInvoiceIsDiscarded(t *testing.T) {
	t.Parallel()

	discarded := Invoice{Status: InvoiceStatusDiscarded}
	if !discarded.IsDiscarded() {
		t.Fatal("IsDiscarded() = false, want true")
	}
	if (Invoice{Status: InvoiceStatusDraft}).IsDiscarded() {
		t.Fatal("IsDiscarded() = true for draft invoice")
	}
	if (Invoice{Status: InvoiceStatusIssued}).IsDiscarded() {
		t.Fatal("IsDiscarded() = true for issued invoice")
	}
}

func TestInvoiceDiscardHelpers(t *testing.T) {
	t.Parallel()

	rate, err := NewMoney(10000, "USD")
	if err != nil {
		t.Fatalf("NewMoney(): %v", err)
	}
	hours, err := NewHours(15000)
	if err != nil {
		t.Fatalf("NewHours(): %v", err)
	}
	entry := TimeEntry{ID: "te_123", Hours: hours}
	line, err := NewInvoiceLine(InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_123", TimeEntryID: entry.ID, UnitRate: rate})
	if err != nil {
		t.Fatalf("NewInvoiceLine(): %v", err)
	}
	invoice, err := NewInvoice(InvoiceParams{CustomerID: "cus_123", Status: InvoiceStatusDraft, Currency: "USD", Lines: []InvoiceLine{line}})
	if err != nil {
		t.Fatalf("NewInvoice(): %v", err)
	}
	if !invoice.IsDraft() {
		t.Fatal("IsDraft() = false, want true")
	}
	if got := invoice.Total([]TimeEntry{entry}); got.Amount == 0 {
		t.Fatal("Total() amount = 0, want positive")
	}
	if _, err := NewMoney(0, "USD"); err == nil {
		t.Fatal("NewMoney() zero amount should fail")
	}
}

func TestNewImportedInvoiceLineAndTotalUseFixedAmount(t *testing.T) {
	t.Parallel()

	line, err := NewImportedInvoiceLine(ImportInvoiceLineParams{Description: "Software development", AmountMinor: 250000, TaxMinor: 0, QuantityDisplay: "160.00", UnitPriceDisplay: "15.6250", Currency: "USD"})
	if err != nil {
		t.Fatalf("NewImportedInvoiceLine() error = %v", err)
	}
	if line.ServiceAgreementID != "" || line.TimeEntryID != "" {
		t.Fatalf("imported line links = (%q,%q), want empty", line.ServiceAgreementID, line.TimeEntryID)
	}
	if line.AmountMinor != 250000 || line.QuantityDisplay != "160.00" || line.UnitPriceDisplay != "15.6250" {
		t.Fatalf("imported line = %+v, want fixed amount and displays", line)
	}
	inv := Invoice{Currency: "USD", Lines: []InvoiceLine{line}}
	if got := inv.Total(nil); got.Amount != 250000 || got.Currency != "USD" {
		t.Fatalf("Total() = %+v, want 250000 USD", got)
	}

	_, err = NewImportedInvoiceLine(ImportInvoiceLineParams{Description: "Refund", AmountMinor: -1, Currency: "USD"})
	if err == nil || !strings.Contains(err.Error(), "amount_minor") {
		t.Fatalf("NewImportedInvoiceLine() error = %v, want amount_minor rejection", err)
	}
}

func TestNewImportedInvoiceValidatesTotalsAndDates(t *testing.T) {
	t.Parallel()

	invoiceDate := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	dueDate := time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC)
	line, err := NewImportedInvoiceLine(ImportInvoiceLineParams{Description: "Software development", AmountMinor: 250000, Currency: "USD"})
	if err != nil {
		t.Fatalf("NewImportedInvoiceLine() error = %v", err)
	}

	inv, err := NewImportedInvoice(ImportInvoiceParams{CustomerID: "cus_1", InvoiceNumber: "INV/2026/00001", InvoiceDate: invoiceDate, DueDate: dueDate, Currency: "USD", PaymentTerms: "15 Days", PaymentCommunication: "INV/2026/00001", ImportSource: "manual-pdf-extract", ExternalNumber: "INV/2026/00001", ImportedAt: time.Date(2026, 4, 26, 13, 25, 34, 0, time.UTC), Lines: []InvoiceLine{line}, SubtotalMinor: 250000, TaxTotalMinor: 0, GrandTotalMinor: 250000})
	if err != nil {
		t.Fatalf("NewImportedInvoice() error = %v", err)
	}
	if inv.Status != InvoiceStatusIssued || inv.InvoiceNumber != "INV/2026/00001" || !inv.IssuedAt.Equal(invoiceDate) || !inv.CreatedAt.Equal(invoiceDate) || !inv.InvoiceDate.Equal(invoiceDate) || !inv.DueDate.Equal(dueDate) {
		t.Fatalf("imported invoice dates/status = %+v, want issued and preserved dates", inv)
	}
	if inv.PaymentTerms != "15 Days" || inv.PaymentCommunication != "INV/2026/00001" || inv.ImportSource != "manual-pdf-extract" || inv.ExternalNumber != "INV/2026/00001" {
		t.Fatalf("import metadata = %+v, want preserved fields", inv)
	}

	_, err = NewImportedInvoice(ImportInvoiceParams{CustomerID: "cus_1", InvoiceNumber: "INV/2026/00002", InvoiceDate: invoiceDate, DueDate: dueDate, Currency: "USD", Lines: []InvoiceLine{line}, SubtotalMinor: 200000, GrandTotalMinor: 200000})
	if err == nil || !strings.Contains(err.Error(), "totals do not balance") {
		t.Fatalf("NewImportedInvoice() error = %v, want totals do not balance", err)
	}
}
