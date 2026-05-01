package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

func TestInvoiceStoreCreateDraft(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID:  customerID,
		Status:      core.InvoiceStatusDraft,
		Currency:    "USD",
		Lines:       []core.InvoiceLine{line},
		PeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC),
		DueDate:     time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		Notes:       "Net 15",
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.InvoiceStatusDraft {
		t.Fatalf("Status = %q, want draft", got.Status)
	}
	if got.CustomerID != customerID {
		t.Fatalf("CustomerID = %q, want %q", got.CustomerID, customerID)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1", len(got.Lines))
	}
	if got.Lines[0].TimeEntryID != entry.ID {
		t.Fatalf("Line TimeEntryID = %q, want %q", got.Lines[0].TimeEntryID, entry.ID)
	}
	if !got.PeriodStart.Equal(invoice.PeriodStart) || !got.PeriodEnd.Equal(invoice.PeriodEnd) || !got.DueDate.Equal(invoice.DueDate) || got.Notes != "Net 15" {
		t.Fatalf("metadata = (%s,%s,%s,%q), want (%s,%s,%s,%q)", got.PeriodStart, got.PeriodEnd, got.DueDate, got.Notes, invoice.PeriodStart, invoice.PeriodEnd, invoice.DueDate, invoice.Notes)
	}
}

func TestInvoiceStoreGetByIDPreMetadataRow(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()
	customerID, _ := seedCustomerAndAgreement(t, store)
	now := time.Now().UTC().UnixNano()
	_, err = store.DB().Exec(`INSERT INTO invoices (id, invoice_number, customer_id, status, currency, created_at, updated_at) VALUES ('inv_legacy', '', ?, 'draft', 'USD', ?, ?)`, customerID, now, now)
	if err != nil {
		t.Fatalf("insert legacy invoice: %v", err)
	}

	got, err := NewInvoiceStore(store).GetByID(context.Background(), "inv_legacy")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if !got.PeriodStart.IsZero() || !got.PeriodEnd.IsZero() || !got.DueDate.IsZero() || got.Notes != "" {
		t.Fatalf("legacy metadata = (%s,%s,%s,%q), want zero dates and empty notes", got.PeriodStart, got.PeriodEnd, got.DueDate, got.Notes)
	}
}

func TestInvoiceStoreGetByID_NotFound(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	_, err = invStore.GetByID(context.Background(), "inv_nonexistent")
	if err == nil {
		t.Fatal("GetByID() error = nil, want not found")
	}
}

func TestInvoiceStoreUpdate(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.InvoiceStatusIssued {
		t.Fatalf("Status = %q, want issued", got.Status)
	}
	if got.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", got.InvoiceNumber)
	}
}

func TestInvoiceStoreUpdateMetadata(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)
	entry := &core.TimeEntry{ID: "te_001", ServiceAgreementID: agreementID, CustomerProfileID: customerID, Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}
	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: agreementID, TimeEntryID: entry.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: customerID, Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line}, PeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), Notes: "Old notes", CreatedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)})
	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	before, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID(before) error = %v", err)
	}

	if err := invoice.UpdateMetadata(core.InvoiceMetadataPatch{InvoiceDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), PaymentTerms: "Net 20", PaymentCommunication: "Use updated reference", Notes: "Updated notes", ExternalNumber: "EXT-UPDATED"}); err != nil {
		t.Fatalf("UpdateMetadata(core) error = %v", err)
	}
	if err := invStore.UpdateMetadata(context.Background(), &invoice); err != nil {
		t.Fatalf("UpdateMetadata() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID(after) error = %v", err)
	}
	if got.InvoiceDate.Format(time.RFC3339) != "2026-05-01T00:00:00Z" || got.DueDate.Format(time.RFC3339) != "2026-05-20T00:00:00Z" || got.PaymentTerms != "Net 20" || got.PaymentCommunication != "Use updated reference" || got.Notes != "Updated notes" || got.ExternalNumber != "EXT-UPDATED" {
		t.Fatalf("metadata after UpdateMetadata = %+v, want updated listed fields", got)
	}
	if got.InvoiceNumber != before.InvoiceNumber || !got.IssuedAt.Equal(before.IssuedAt) || got.Status != before.Status || got.CustomerID != before.CustomerID {
		t.Fatalf("non-metadata invoice fields changed: before=%+v after=%+v", before, got)
	}
	if len(got.Lines) != 1 || got.Lines[0].ID != before.Lines[0].ID || got.Total(nil).Amount != before.Total(nil).Amount {
		t.Fatalf("lines/totals changed: before=%+v total=%+v after=%+v total=%+v", before.Lines, before.Total(nil), got.Lines, got.Total(nil))
	}
	if !got.UpdatedAt.After(before.UpdatedAt) {
		t.Fatalf("UpdatedAt = %s, want after %s", got.UpdatedAt, before.UpdatedAt)
	}
}

func TestInvoiceStoreHealthChecks(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	if err := invStore.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	version, err := invStore.SchemaVersion(context.Background())
	if err != nil {
		t.Fatalf("SchemaVersion() error = %v", err)
	}
	if version < 0 {
		t.Fatalf("SchemaVersion() = %d, want non-negative", version)
	}
}

func TestInvoiceStoreDelete(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	if err := invStore.Delete(context.Background(), invoice.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = invStore.GetByID(context.Background(), invoice.ID)
	if err == nil {
		t.Fatal("GetByID() after delete error = nil, want not found")
	}

	// Time entry should be unlocked (invoice_id = NULL).
	teStore := NewTimeEntryStore(store)
	gotEntry, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after delete: %v", err)
	}
	if gotEntry.InvoiceID != "" {
		t.Fatalf("Entry InvoiceID = %q, want empty after draft delete", gotEntry.InvoiceID)
	}
}

func TestInvoiceStoreSoftDiscard(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	// Issue the invoice.
	issuedAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	if err := invoice.Issue("INV-2026-0001", issuedAt); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Soft-discard (issued path).
	discardedAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	if err := invoice.Discard(discardedAt); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice); err != nil {
		t.Fatalf("Update() soft-discard error = %v", err)
	}

	// Invoice should still exist with status=discarded.
	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.InvoiceStatusDiscarded {
		t.Fatalf("Status = %q, want discarded", got.Status)
	}
	if got.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", got.InvoiceNumber)
	}

	// Time entry should still be locked to the invoice.
	teStore := NewTimeEntryStore(store)
	gotEntry, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after soft-discard: %v", err)
	}
	if gotEntry.InvoiceID != invoice.ID {
		t.Fatalf("Entry InvoiceID = %q, want %q (should remain locked)", gotEntry.InvoiceID, invoice.ID)
	}
}

func TestInvoiceStoreSoftDiscard_NextIssueGetsNextNumber(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invStore := NewInvoiceStore(store)
	seqStore := NewInvoiceSequenceStore(store)

	// Create and issue first invoice.
	invoice1, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})
	if err := invStore.CreateDraft(context.Background(), &invoice1, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	num1, err := seqStore.Next(context.Background())
	if err != nil {
		t.Fatalf("first Next() error = %v", err)
	}
	if err := invoice1.Issue(num1, time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice1); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if num1 != "INV-2026-0001" {
		t.Fatalf("first number = %q, want INV-2026-0001", num1)
	}

	// Soft-discard the issued invoice.
	if err := invoice1.Discard(time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice1); err != nil {
		t.Fatalf("soft-discard Update() error = %v", err)
	}

	// Create and issue second invoice — must get INV-2026-0002, not reuse 0001.
	entry2 := &core.TimeEntry{
		ID:                 "te_002",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work 2",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry2); err != nil {
		t.Fatalf("save entry2: %v", err)
	}
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry2.ID,
		UnitRate:           rate,
	})
	invoice2, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line2},
	})
	if err := invStore.CreateDraft(context.Background(), &invoice2, []*core.TimeEntry{entry2}); err != nil {
		t.Fatalf("CreateDraft() second error = %v", err)
	}
	num2, err := seqStore.Next(context.Background())
	if err != nil {
		t.Fatalf("second Next() error = %v", err)
	}
	if num2 != "INV-2026-0002" {
		t.Fatalf("second number = %q, want INV-2026-0002 (must not reuse 0001)", num2)
	}
	if err := invoice2.Issue(num2, time.Date(2026, 4, 12, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Issue() second error = %v", err)
	}
	if err := invStore.Update(context.Background(), &invoice2); err != nil {
		t.Fatalf("Update() second error = %v", err)
	}

	// Verify first invoice still exists as discarded.
	got1, err := invStore.GetByID(context.Background(), invoice1.ID)
	if err != nil {
		t.Fatalf("GetByID() first error = %v", err)
	}
	if got1.Status != core.InvoiceStatusDiscarded {
		t.Fatalf("first invoice status = %q, want discarded", got1.Status)
	}
	if got1.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("first invoice number = %q, want INV-2026-0001", got1.InvoiceNumber)
	}
}

func TestInvoiceStoreDelete_IsAtomic(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry := &core.TimeEntry{
		ID:                 "te_001",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	// Verify entry is locked.
	teStore := NewTimeEntryStore(store)
	gotEntry, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry before delete: %v", err)
	}
	if gotEntry.InvoiceID != invoice.ID {
		t.Fatalf("Entry InvoiceID = %q, want %q (should be locked)", gotEntry.InvoiceID, invoice.ID)
	}

	// Delete should unlock entries and remove invoice atomically.
	if err := invStore.Delete(context.Background(), invoice.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify invoice is gone.
	_, err = invStore.GetByID(context.Background(), invoice.ID)
	if err == nil {
		t.Fatal("GetByID() after delete error = nil, want not found")
	}

	// Verify entry is unlocked.
	gotEntry, err = teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after delete: %v", err)
	}
	if gotEntry.InvoiceID != "" {
		t.Fatalf("Entry InvoiceID = %q, want empty (should be unlocked)", gotEntry.InvoiceID)
	}
}

func TestInvoiceStoreAddLineRemoveLineAndSnapshotTotals(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()
	customerID, agreementID := seedCustomerAndAgreement(t, store)
	teStore := NewTimeEntryStore(store)
	entry := &core.TimeEntry{ID: "te_edit_001", ServiceAgreementID: agreementID, CustomerProfileID: customerID, Description: "Original work", Hours: mustHours(12000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := teStore.Save(context.Background(), entry); err != nil {
		t.Fatalf("save entry: %v", err)
	}
	rate, _ := core.NewMoney(6000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: agreementID, TimeEntryID: entry.ID, UnitRate: rate, Description: entry.Description, QuantityMin: 72})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: customerID, Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line}})
	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	manual, _ := core.NewManualInvoiceLine(invoice.ID, agreementID, "Manual setup", 60, core.Money{Amount: 3000, Currency: "USD"}, "USD")
	if err := invStore.AddLine(context.Background(), invoice.ID, manual); err != nil {
		t.Fatalf("AddLine() error = %v", err)
	}
	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(got.Lines) != 2 || got.Lines[1].TimeEntryID != "" || got.Lines[1].Description != "Manual setup" || got.Lines[1].QuantityMin != 60 {
		t.Fatalf("manual line after GetByID = %+v", got.Lines)
	}
	summaries, err := invStore.ListByCustomer(context.Background(), customerID)
	if err != nil {
		t.Fatalf("ListByCustomer() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].GrandTotal != 10200 {
		t.Fatalf("snapshot summary = %+v, want total 10200", summaries)
	}
	if err := invStore.RemoveLine(context.Background(), invoice.ID, line.ID); err != nil {
		t.Fatalf("RemoveLine() error = %v", err)
	}
	entryAfter, err := teStore.GetByID(context.Background(), entry.ID)
	if err != nil {
		t.Fatalf("Get entry after RemoveLine: %v", err)
	}
	if entryAfter.InvoiceID != "" {
		t.Fatalf("entry InvoiceID = %q, want unlocked", entryAfter.InvoiceID)
	}
	got, _ = invStore.GetByID(context.Background(), invoice.ID)
	if len(got.Lines) != 1 || got.Lines[0].ID != manual.ID {
		t.Fatalf("lines after remove = %+v, want only manual", got.Lines)
	}
}

func TestInvoiceStoreImportIssuedRoundTripAndDuplicate(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()
	customerID, _ := seedCustomerAndAgreement(t, store)
	invoice := importedInvoiceForStoreTest(t, customerID, "INV/2026/00001")
	invStore := NewInvoiceStore(store)
	if err := invStore.ImportIssued(context.Background(), &invoice); err != nil {
		t.Fatalf("ImportIssued() error = %v", err)
	}
	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.InvoiceNumber != "INV/2026/00001" || got.Status != core.InvoiceStatusIssued || !got.InvoiceDate.Equal(invoice.InvoiceDate) || got.PaymentCommunication != "INV/2026/00001" || got.Lines[0].AmountMinor != 250000 || got.Lines[0].QuantityDisplay != "160.00" {
		t.Fatalf("round trip invoice = %+v line=%+v, want imported fields", got, got.Lines[0])
	}
	summaries, err := invStore.ListByCustomer(context.Background(), customerID, core.InvoiceStatusIssued)
	if err != nil {
		t.Fatalf("ListByCustomer() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].GrandTotal != 250000 {
		t.Fatalf("summaries = %+v, want imported grand total", summaries)
	}
	duplicate := importedInvoiceForStoreTest(t, customerID, "INV/2026/00001")
	if err := invStore.ImportIssued(context.Background(), &duplicate); err == nil {
		t.Fatal("ImportIssued(duplicate) error = nil, want duplicate error")
	}

	var timeEntryCount int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM time_entries WHERE invoice_id = ?`, invoice.ID).Scan(&timeEntryCount); err != nil {
		t.Fatalf("count time entries: %v", err)
	}
	if timeEntryCount != 0 {
		t.Fatalf("time entries touched = %d, want 0", timeEntryCount)
	}
}

func TestInvoiceStoreListByCustomerIncludesImportedTax(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()
	customerID, _ := seedCustomerAndAgreement(t, store)
	line, err := core.NewImportedInvoiceLine(core.ImportInvoiceLineParams{Description: "Taxable service", AmountMinor: 100000, TaxMinor: 21000, QuantityDisplay: "1.00", UnitPriceDisplay: "1000.00", Currency: "USD"})
	if err != nil {
		t.Fatalf("NewImportedInvoiceLine(): %v", err)
	}
	invoice, err := core.NewImportedInvoice(core.ImportInvoiceParams{CustomerID: customerID, InvoiceNumber: "INV/2026/TAX", InvoiceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), Currency: "USD", Lines: []core.InvoiceLine{line}, SubtotalMinor: 100000, TaxTotalMinor: 21000, GrandTotalMinor: 121000})
	if err != nil {
		t.Fatalf("NewImportedInvoice(): %v", err)
	}
	invStore := NewInvoiceStore(store)
	if err := invStore.ImportIssued(context.Background(), &invoice); err != nil {
		t.Fatalf("ImportIssued() error = %v", err)
	}

	summaries, err := invStore.ListByCustomer(context.Background(), customerID, core.InvoiceStatusIssued)
	if err != nil {
		t.Fatalf("ListByCustomer() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].GrandTotal != 121000 {
		t.Fatalf("summaries = %+v, want imported grand total including tax 121000", summaries)
	}
}

func TestInvoiceStoreImportIssuedRollsBackLineInsertFailure(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()
	customerID, _ := seedCustomerAndAgreement(t, store)
	line1, err := core.NewImportedInvoiceLine(core.ImportInvoiceLineParams{Description: "First line", AmountMinor: 60000, Currency: "USD"})
	if err != nil {
		t.Fatalf("NewImportedInvoiceLine(line1): %v", err)
	}
	line2, err := core.NewImportedInvoiceLine(core.ImportInvoiceLineParams{Description: "Second line", AmountMinor: 40000, Currency: "USD"})
	if err != nil {
		t.Fatalf("NewImportedInvoiceLine(line2): %v", err)
	}
	line2.ID = line1.ID
	invoice, err := core.NewImportedInvoice(core.ImportInvoiceParams{CustomerID: customerID, InvoiceNumber: "INV/2026/ROLLBACK", InvoiceDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), Currency: "USD", Lines: []core.InvoiceLine{line1, line2}, SubtotalMinor: 100000, GrandTotalMinor: 100000})
	if err != nil {
		t.Fatalf("NewImportedInvoice(): %v", err)
	}
	invStore := NewInvoiceStore(store)
	err = invStore.ImportIssued(context.Background(), &invoice)
	if err == nil {
		t.Fatal("ImportIssued() error = nil, want duplicate line id insert failure")
	}

	var invoiceRows, lineRows int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM invoices WHERE id = ?`, invoice.ID).Scan(&invoiceRows); err != nil {
		t.Fatalf("count invoices: %v", err)
	}
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM invoice_lines WHERE invoice_id = ?`, invoice.ID).Scan(&lineRows); err != nil {
		t.Fatalf("count invoice lines: %v", err)
	}
	if invoiceRows != 0 || lineRows != 0 {
		t.Fatalf("rolled back rows = invoices:%d lines:%d, want both 0", invoiceRows, lineRows)
	}
}

func importedInvoiceForStoreTest(t *testing.T, customerID, number string) core.Invoice {
	t.Helper()
	line, err := core.NewImportedInvoiceLine(core.ImportInvoiceLineParams{Description: "Software development", AmountMinor: 250000, QuantityDisplay: "160.00", UnitPriceDisplay: "15.6250", Currency: "USD"})
	if err != nil {
		t.Fatalf("NewImportedInvoiceLine(): %v", err)
	}
	invoice, err := core.NewImportedInvoice(core.ImportInvoiceParams{CustomerID: customerID, InvoiceNumber: number, InvoiceDate: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 2, 17, 0, 0, 0, 0, time.UTC), Currency: "USD", PaymentTerms: "15 Days", PaymentCommunication: number, ImportSource: "manual-pdf-extract", ExternalNumber: number, ImportedAt: time.Date(2026, 4, 26, 13, 25, 34, 0, time.UTC), Lines: []core.InvoiceLine{line}, SubtotalMinor: 250000, GrandTotalMinor: 250000})
	if err != nil {
		t.Fatalf("NewImportedInvoice(): %v", err)
	}
	return invoice
}

// seedCustomerAndAgreement creates a minimal customer profile and service agreement for testing.
func seedCustomerAndAgreement(t *testing.T, store *Store) (customerID, agreementID string) {
	t.Helper()

	db := store.DB()

	// Create legal entity.
	_, err := db.Exec(`INSERT INTO legal_entities (id, type, legal_name, created_at, updated_at) VALUES (?, 'company', 'Test Co', ?, ?)`,
		"le_test", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("seed legal entity: %v", err)
	}

	// Create customer profile.
	_, err = db.Exec(`INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, created_at, updated_at) VALUES (?, ?, 'active', 'USD', ?, ?)`,
		"cus_test", "le_test", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	// Create service agreement.
	_, err = db.Exec(`INSERT INTO service_agreements (id, customer_profile_id, name, billing_mode, hourly_rate, currency, active, created_at, updated_at) VALUES (?, ?, 'Support', 'hourly', 10000, 'USD', 1, ?, ?)`,
		"sa_test", "cus_test", time.Now().UTC().UnixNano(), time.Now().UTC().UnixNano())
	if err != nil {
		t.Fatalf("seed agreement: %v", err)
	}

	return "cus_test", "sa_test"
}

func TestNewInvoiceStore_NilStore(t *testing.T) {
	t.Parallel()

	got := NewInvoiceStore(nil)
	if got != nil {
		t.Fatalf("NewInvoiceStore(nil) = %v, want nil", got)
	}
}

func TestInvoiceStoreCreateDraft_NilInvoice(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	err = invStore.CreateDraft(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("CreateDraft(nil) error = nil, want error")
	}
}

func TestInvoiceStoreUpdate_NilInvoice(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	err = invStore.Update(context.Background(), nil)
	if err == nil {
		t.Fatal("Update(nil) error = nil, want error")
	}
}

func TestInvoiceStoreGetByID_MultipleLines(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	entry1 := &core.TimeEntry{
		ID:                 "te_ml_1",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work 1",
		Hours:              mustHours(15000),
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	entry2 := &core.TimeEntry{
		ID:                 "te_ml_2",
		ServiceAgreementID: agreementID,
		CustomerProfileID:  customerID,
		Description:        "Work 2",
		Hours:              mustHours(20000),
		Billable:           true,
		Date:               time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	teStore := NewTimeEntryStore(store)
	for _, e := range []*core.TimeEntry{entry1, entry2} {
		if err := teStore.Save(context.Background(), e); err != nil {
			t.Fatalf("save entry: %v", err)
		}
	}

	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_ml_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry1.ID,
		UnitRate:           rate,
	})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{
		InvoiceID:          "inv_ml_seed",
		ServiceAgreementID: agreementID,
		TimeEntryID:        entry2.ID,
		UnitRate:           rate,
	})

	invoice, _ := core.NewInvoice(core.InvoiceParams{
		CustomerID: customerID,
		Status:     core.InvoiceStatusDraft,
		Currency:   "USD",
		Lines:      []core.InvoiceLine{line1, line2},
	})

	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &invoice, []*core.TimeEntry{entry1, entry2}); err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}

	got, err := invStore.GetByID(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if len(got.Lines) != 2 {
		t.Fatalf("len(Lines) = %d, want 2", len(got.Lines))
	}
}

func TestInvoiceStoreDelete_AlreadyGone(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	// Deleting a non-existent invoice should not error (SQL UPDATE/DELETE with 0 rows is fine).
	if err := invStore.Delete(context.Background(), "inv_does_not_exist"); err != nil {
		t.Fatalf("Delete(nonexistent) error = %v, want nil", err)
	}
}

func TestInvoiceStoreListByCustomer(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	customerID, agreementID := seedCustomerAndAgreement(t, store)

	// Create two time entries and two invoices
	rate, _ := core.NewMoney(10000, "USD")

	entry1 := &core.TimeEntry{
		ID: "te_list_001", ServiceAgreementID: agreementID, CustomerProfileID: customerID,
		Description: "Work A", Hours: mustHours(15000), Billable: true,
		Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry1); err != nil {
		t.Fatalf("save entry1: %v", err)
	}
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: agreementID, TimeEntryID: entry1.ID, UnitRate: rate})
	inv1, _ := core.NewInvoice(core.InvoiceParams{CustomerID: customerID, Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1}, PeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)})
	invStore := NewInvoiceStore(store)
	if err := invStore.CreateDraft(context.Background(), &inv1, []*core.TimeEntry{entry1}); err != nil {
		t.Fatalf("CreateDraft inv1: %v", err)
	}

	entry2 := &core.TimeEntry{
		ID: "te_list_002", ServiceAgreementID: agreementID, CustomerProfileID: customerID,
		Description: "Work B", Hours: mustHours(30000), Billable: true,
		Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := NewTimeEntryStore(store).Save(context.Background(), entry2); err != nil {
		t.Fatalf("save entry2: %v", err)
	}
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: agreementID, TimeEntryID: entry2.ID, UnitRate: rate})
	inv2, _ := core.NewInvoice(core.InvoiceParams{CustomerID: customerID, Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line2}})
	if err := invStore.CreateDraft(context.Background(), &inv2, []*core.TimeEntry{entry2}); err != nil {
		t.Fatalf("CreateDraft inv2: %v", err)
	}

	// List all
	summaries, err := invStore.ListByCustomer(context.Background(), customerID)
	if err != nil {
		t.Fatalf("ListByCustomer() error = %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}
	// grand_total = unit_rate_amount * hours / 10000
	// entry1: 10000 * 15000 / 10000 = 15000; entry2: 10000 * 30000 / 10000 = 30000
	totals := map[string]int64{}
	for _, s := range summaries {
		totals[s.ID] = s.GrandTotal
	}
	if totals[inv1.ID] != 15000 {
		t.Fatalf("inv1 GrandTotal = %d, want 15000", totals[inv1.ID])
	}
	if totals[inv2.ID] != 30000 {
		t.Fatalf("inv2 GrandTotal = %d, want 30000", totals[inv2.ID])
	}
	metadata := map[string]core.InvoiceSummary{}
	for _, s := range summaries {
		metadata[s.ID] = s
	}
	if !metadata[inv1.ID].PeriodStart.Equal(inv1.PeriodStart) || !metadata[inv1.ID].PeriodEnd.Equal(inv1.PeriodEnd) || !metadata[inv1.ID].DueDate.Equal(inv1.DueDate) {
		t.Fatalf("inv1 summary metadata = (%s,%s,%s), want (%s,%s,%s)", metadata[inv1.ID].PeriodStart, metadata[inv1.ID].PeriodEnd, metadata[inv1.ID].DueDate, inv1.PeriodStart, inv1.PeriodEnd, inv1.DueDate)
	}

	// List with status filter = draft → both
	filtered, err := invStore.ListByCustomer(context.Background(), customerID, core.InvoiceStatusDraft)
	if err != nil {
		t.Fatalf("ListByCustomer(draft) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(filtered))
	}

	// List with status filter = issued → none
	issued, err := invStore.ListByCustomer(context.Background(), customerID, core.InvoiceStatusIssued)
	if err != nil {
		t.Fatalf("ListByCustomer(issued) error = %v", err)
	}
	if len(issued) != 0 {
		t.Fatalf("len(issued) = %d, want 0", len(issued))
	}
}

func TestInvoiceStoreListByCustomer_Empty(t *testing.T) {
	t.Parallel()

	store, err := Open("")
	if err != nil {
		t.Fatalf("Open(): %v", err)
	}
	defer store.Close()

	invStore := NewInvoiceStore(store)
	summaries, err := invStore.ListByCustomer(context.Background(), "cus_nonexistent")
	if err != nil {
		t.Fatalf("ListByCustomer() error = %v", err)
	}
	if summaries != nil && len(summaries) != 0 {
		t.Fatalf("len(summaries) = %d, want 0", len(summaries))
	}
}

func mustHours(amount int64) core.Hours {
	h, err := core.NewHours(amount)
	if err != nil {
		panic(err)
	}
	return h
}
