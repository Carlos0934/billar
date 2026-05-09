package app

import (
	"context"
	"errors"
	"testing"

	"github.com/Carlos0934/billar/internal/core"
)

type quoteStoreStub struct {
	createArg      *core.Quote
	getByIDRes     *core.Quote
	getByIDErr     error
	listRes        []core.QuoteSummary
	addLineQuoteID string
	addLineArg     core.QuoteLine
	updated        *core.Quote
	deletedID      string
}

func (s *quoteStoreStub) Create(ctx context.Context, quote *core.Quote) error {
	_ = ctx
	s.createArg = quote
	return nil
}

func (s *quoteStoreStub) GetByID(ctx context.Context, id string) (*core.Quote, error) {
	_ = ctx
	return s.getByIDRes, s.getByIDErr
}

func (s *quoteStoreStub) ListByCustomer(ctx context.Context, customerID string, status ...core.QuoteStatus) ([]core.QuoteSummary, error) {
	_ = ctx
	return s.listRes, nil
}

func (s *quoteStoreStub) AddLine(ctx context.Context, quoteID string, line core.QuoteLine) error {
	_ = ctx
	s.addLineQuoteID = quoteID
	s.addLineArg = line
	return nil
}

func (s *quoteStoreStub) Update(ctx context.Context, quote *core.Quote) error {
	_ = ctx
	s.updated = quote
	return nil
}

func (s *quoteStoreStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deletedID = id
	return nil
}

func TestQuoteServiceCreateValidatesActiveCustomer(t *testing.T) {
	t.Parallel()

	profiles := &customerProfileStoreForAgreements{getByIDRes: &core.CustomerProfile{ID: "cus_123", Status: core.CustomerProfileStatusActive, DefaultCurrency: "USD"}}
	quotes := &quoteStoreStub{}
	svc := NewQuoteService(quotes, profiles, nil)

	dto, err := svc.Create(context.Background(), CreateQuoteCommand{CustomerID: "cus_123", Currency: "USD", Notes: "offer"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if quotes.createArg == nil {
		t.Fatal("QuoteStore.Create was not called")
	}
	if dto.Status != "draft" || dto.CustomerID != "cus_123" || dto.Currency != "USD" {
		t.Fatalf("Create() DTO = %+v, want draft quote for cus_123 USD", dto)
	}

	inactive := &customerProfileStoreForAgreements{getByIDRes: &core.CustomerProfile{ID: "cus_inactive", Status: core.CustomerProfileStatusInactive, DefaultCurrency: "USD"}}
	_, err = NewQuoteService(&quoteStoreStub{}, inactive, nil).Create(context.Background(), CreateQuoteCommand{CustomerID: "cus_inactive", Currency: "USD"})
	if !errors.Is(err, ErrCustomerProfileInactive) {
		t.Fatalf("Create() inactive error = %v, want ErrCustomerProfileInactive", err)
	}
}

func TestQuoteServiceListAndShow(t *testing.T) {
	t.Parallel()

	quote := &core.Quote{ID: "quo_123", CustomerID: "cus_123", Status: core.QuoteStatusSent, Currency: "USD", Lines: []core.QuoteLine{{ID: "qol_1", QuoteID: "quo_123", ServiceAgreementID: "sa_123", Description: "Work", QuantityMin: 60, UnitRate: core.Money{Amount: 2500, Currency: "USD"}}}}
	quotes := &quoteStoreStub{getByIDRes: quote, listRes: []core.QuoteSummary{{ID: "quo_123", CustomerID: "cus_123", Status: core.QuoteStatusSent, Currency: "USD", Total: core.Money{Amount: 2500, Currency: "USD"}}}}
	svc := NewQuoteService(quotes, nil, nil)

	shown, err := svc.Get(context.Background(), "quo_123")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if shown.Total != 2500 || len(shown.Lines) != 1 || shown.Lines[0].LineTotalAmount != 2500 {
		t.Fatalf("Get() DTO = %+v, want line-derived total 2500", shown)
	}
	listed, err := svc.List(context.Background(), ListQuotesCommand{CustomerID: "cus_123", Status: "sent"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].Total != 2500 || listed[0].Status != "sent" {
		t.Fatalf("List() = %+v, want sent total 2500", listed)
	}

	missing := NewQuoteService(&quoteStoreStub{getByIDErr: ErrQuoteNotFound}, nil, nil)
	_, err = missing.Get(context.Background(), "quo_missing")
	if !errors.Is(err, ErrQuoteNotFound) {
		t.Fatalf("Get() missing error = %v, want ErrQuoteNotFound", err)
	}
}

func TestQuoteServiceAddLineValidatesAgreement(t *testing.T) {
	t.Parallel()

	quote := &core.Quote{ID: "quo_123", CustomerID: "cus_123", Status: core.QuoteStatusDraft, Currency: "USD"}
	agreements := &serviceAgreementStoreStub{getByIDRes: &core.ServiceAgreement{ID: "sa_123", CustomerProfileID: "cus_123", Active: true, Currency: "USD", HourlyRate: 2500}}
	quotes := &quoteStoreStub{getByIDRes: quote}
	svc := NewQuoteService(quotes, nil, agreements)

	dto, err := svc.AddLine(context.Background(), AddQuoteLineCommand{QuoteID: "quo_123", ServiceAgreementID: "sa_123", Description: "Work", QuantityMin: 60})
	if err != nil {
		t.Fatalf("AddLine() error = %v", err)
	}
	if quotes.addLineQuoteID != "quo_123" || quotes.addLineArg.UnitRate.Amount != 2500 {
		t.Fatalf("AddLine stored quoteID=%q line=%+v, want rate snapshot 2500", quotes.addLineQuoteID, quotes.addLineArg)
	}
	if dto.Total != 2500 || len(dto.Lines) != 1 {
		t.Fatalf("AddLine() DTO = %+v, want total 2500 with one line", dto)
	}

	mismatch := &serviceAgreementStoreStub{getByIDRes: &core.ServiceAgreement{ID: "sa_bad", CustomerProfileID: "other", Active: true, Currency: "USD", HourlyRate: 2500}}
	_, err = NewQuoteService(&quoteStoreStub{getByIDRes: quote}, nil, mismatch).AddLine(context.Background(), AddQuoteLineCommand{QuoteID: "quo_123", ServiceAgreementID: "sa_bad", Description: "Work", QuantityMin: 60})
	if err == nil {
		t.Fatal("AddLine() mismatch error = nil, want error")
	}
}

func TestQuoteServiceLifecycleDeleteAndEligibility(t *testing.T) {
	t.Parallel()

	quote := &core.Quote{ID: "quo_123", CustomerID: "cus_123", Status: core.QuoteStatusSent, Currency: "USD"}
	quotes := &quoteStoreStub{getByIDRes: quote}
	svc := NewQuoteService(quotes, nil, nil)

	accepted, err := svc.Accept(context.Background(), "quo_123")
	if err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	if accepted.Status != "accepted" || !accepted.CanConvertToInvoice || quotes.updated.Status != core.QuoteStatusAccepted {
		t.Fatalf("Accept() DTO=%+v updated=%+v, want accepted convertible", accepted, quotes.updated)
	}
	if err := svc.Delete(context.Background(), "quo_123"); !errors.Is(err, core.ErrAcceptedQuoteCannotBeDeleted) {
		t.Fatalf("Delete() accepted error = %v, want accepted delete guard", err)
	}

	draftStore := &quoteStoreStub{getByIDRes: &core.Quote{ID: "quo_draft", CustomerID: "cus_123", Status: core.QuoteStatusDraft, Currency: "USD"}}
	if err := NewQuoteService(draftStore, nil, nil).Delete(context.Background(), "quo_draft"); err != nil {
		t.Fatalf("Delete() draft error = %v", err)
	}
	if draftStore.deletedID != "quo_draft" {
		t.Fatalf("deletedID = %q, want quo_draft", draftStore.deletedID)
	}
}
