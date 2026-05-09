package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

func TestQuoteStoreCreatePersistsQuoteAndLinesTransactionally(t *testing.T) {
	t.Parallel()

	store := newQuoteTestStore(t)
	quoteStore := NewQuoteStore(store)
	quote := quoteFixture(t, "quo_create", core.QuoteStatusDraft)
	quote.Lines = []core.QuoteLine{quoteLineFixture("qol_create", quote.ID, 90, 12000)}

	if err := quoteStore.Create(context.Background(), quote); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := quoteStore.GetByID(context.Background(), quote.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != quote.ID || got.CustomerID != quote.CustomerID || got.Status != core.QuoteStatusDraft || got.Currency != "USD" {
		t.Fatalf("quote = %+v, want id/customer/status/currency from fixture", got)
	}
	if len(got.Lines) != 1 {
		t.Fatalf("len(lines) = %d, want 1", len(got.Lines))
	}
	if got.Lines[0].LineTotal().Amount != 18000 {
		t.Fatalf("line total = %d, want 18000", got.Lines[0].LineTotal().Amount)
	}
}

func TestQuoteStoreAddLineUpdatesLineDerivedTotalAndListsByFilter(t *testing.T) {
	t.Parallel()

	store := newQuoteTestStore(t)
	quoteStore := NewQuoteStore(store)
	draft := quoteFixture(t, "quo_draft", core.QuoteStatusDraft)
	accepted := quoteFixture(t, "quo_accepted", core.QuoteStatusAccepted)
	if err := quoteStore.Create(context.Background(), draft); err != nil {
		t.Fatalf("Create(draft) error = %v", err)
	}
	if err := quoteStore.Create(context.Background(), accepted); err != nil {
		t.Fatalf("Create(accepted) error = %v", err)
	}

	if err := quoteStore.AddLine(context.Background(), draft.ID, quoteLineFixture("qol_added_1", draft.ID, 60, 5000)); err != nil {
		t.Fatalf("AddLine(first) error = %v", err)
	}
	if err := quoteStore.AddLine(context.Background(), draft.ID, quoteLineFixture("qol_added_2", draft.ID, 30, 8000)); err != nil {
		t.Fatalf("AddLine(second) error = %v", err)
	}

	summaries, err := quoteStore.ListByCustomer(context.Background(), "cus_quote", core.QuoteStatusDraft)
	if err != nil {
		t.Fatalf("ListByCustomer() error = %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1 draft quote", len(summaries))
	}
	if summaries[0].ID != draft.ID || summaries[0].Total.Amount != 9000 {
		t.Fatalf("summary = %+v, want draft total 9000", summaries[0])
	}
}

func TestQuoteStoreQuoteDoesNotAppearInInvoiceStore(t *testing.T) {
	t.Parallel()

	store := newQuoteTestStore(t)
	quoteStore := NewQuoteStore(store)
	invoiceStore := NewInvoiceStore(store)
	quote := quoteFixture(t, "quo_not_invoice", core.QuoteStatusAccepted)
	quote.Lines = []core.QuoteLine{quoteLineFixture("qol_not_invoice", quote.ID, 45, 12000)}

	if err := quoteStore.Create(context.Background(), quote); err != nil {
		t.Fatalf("Create(quote) error = %v", err)
	}

	invoices, err := invoiceStore.ListByCustomer(context.Background(), quote.CustomerID)
	if err != nil {
		t.Fatalf("ListByCustomer(invoices) error = %v", err)
	}
	if len(invoices) != 0 {
		t.Fatalf("invoice summaries = %+v, want no invoices for persisted quote", invoices)
	}

	_, err = invoiceStore.GetByID(context.Background(), quote.ID)
	if !errors.Is(err, app.ErrInvoiceNotFound) {
		t.Fatalf("GetByID(quote ID as invoice) error = %v, want ErrInvoiceNotFound", err)
	}
}

func TestQuoteStoreUpdateDeleteAndNotFound(t *testing.T) {
	t.Parallel()

	store := newQuoteTestStore(t)
	quoteStore := NewQuoteStore(store)
	quote := quoteFixture(t, "quo_lifecycle", core.QuoteStatusSent)
	if err := quoteStore.Create(context.Background(), quote); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	quote.Status = core.QuoteStatusAccepted
	quote.AcceptedAt = time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	quote.UpdatedAt = quote.AcceptedAt
	if err := quoteStore.Update(context.Background(), quote); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	got, err := quoteStore.GetByID(context.Background(), quote.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Status != core.QuoteStatusAccepted || !got.AcceptedAt.Equal(quote.AcceptedAt) {
		t.Fatalf("updated quote = %+v, want accepted with timestamp", got)
	}
	if err := quoteStore.Delete(context.Background(), quote.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = quoteStore.GetByID(context.Background(), quote.ID)
	if !errors.Is(err, app.ErrQuoteNotFound) {
		t.Fatalf("GetByID(deleted) error = %v, want ErrQuoteNotFound", err)
	}
}

func newQuoteTestStore(t *testing.T) *Store {
	t.Helper()
	store := newTestStore(t)
	newAgreementFixture(t, store.DB(), "le_quote", "cus_quote")
	insertServiceAgreement(t, store.DB(), core.ServiceAgreement{ID: "sa_quote", CustomerProfileID: "cus_quote", Name: "Quote Support", BillingMode: core.BillingModeHourly, HourlyRate: 12000, Currency: "USD", Active: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	return store
}

func quoteFixture(t *testing.T, id string, status core.QuoteStatus) *core.Quote {
	t.Helper()
	now := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	return &core.Quote{ID: id, CustomerID: "cus_quote", Status: status, Currency: "USD", Notes: "Prepared quote", CreatedAt: now, UpdatedAt: now}
}

func quoteLineFixture(id, quoteID string, quantityMin, hourlyRate int64) core.QuoteLine {
	return core.QuoteLine{ID: id, QuoteID: quoteID, ServiceAgreementID: "sa_quote", Description: "Consulting", QuantityMin: quantityMin, UnitRate: core.Money{Amount: hourlyRate, Currency: "USD"}}
}
