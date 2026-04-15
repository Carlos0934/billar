package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrNoUnbilledEntries = errors.New("no unbilled time entries")
var ErrCustomerProfileInactive = errors.New("customer profile is inactive")
var ErrInvoiceNotFound = errors.New("invoice not found")

type InvoiceStore interface {
	CreateDraft(ctx context.Context, invoice *core.Invoice, entries []*core.TimeEntry) error
	GetByID(ctx context.Context, id string) (*core.Invoice, error)
	Update(ctx context.Context, invoice *core.Invoice) error
	Delete(ctx context.Context, id string) error
}

type InvoiceNumberGenerator interface {
	Next(ctx context.Context) (string, error)
}

type InvoiceService struct {
	invoices   InvoiceStore
	entries    TimeEntryStore
	agreements ServiceAgreementStore
	profiles   CustomerProfileStore
	numbers    InvoiceNumberGenerator
}

func NewInvoiceService(invoices InvoiceStore, entries TimeEntryStore, agreements ServiceAgreementStore, profiles CustomerProfileStore, numbers ...InvoiceNumberGenerator) InvoiceService {
	var numberGen InvoiceNumberGenerator
	if len(numbers) > 0 {
		numberGen = numbers[0]
	}
	return InvoiceService{invoices: invoices, entries: entries, agreements: agreements, profiles: profiles, numbers: numberGen}
}

func (s InvoiceService) CreateDraftFromUnbilled(ctx context.Context, cmd CreateDraftFromUnbilledCommand) (InvoiceDTO, error) {
	if strings.TrimSpace(cmd.CustomerProfileID) == "" {
		return InvoiceDTO{}, errors.New("customer profile id is required")
	}
	if s.profiles == nil || s.entries == nil || s.agreements == nil || s.invoices == nil {
		return InvoiceDTO{}, errors.New("invoice service dependencies are required")
	}

	profile, err := s.getCustomerProfile(ctx, cmd.CustomerProfileID)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", err)
	}
	if profile == nil {
		return InvoiceDTO{}, errors.New("create invoice draft: customer profile is required")
	}
	if !profile.CanReceiveInvoices() {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", ErrCustomerProfileInactive)
	}

	entries, err := s.entries.ListUnbilled(ctx, cmd.CustomerProfileID)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: list unbilled entries: %w", err)
	}
	if len(entries) == 0 {
		return InvoiceDTO{}, ErrNoUnbilledEntries
	}

	lockedEntries := make([]*core.TimeEntry, 0, len(entries))
	lines := make([]core.InvoiceLine, 0, len(entries))
	for i := range entries {
		entry := entries[i]
		if entry.CustomerProfileID != cmd.CustomerProfileID {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: time entry customer mismatch for %s", entry.ID)
		}
		agreement, err := s.getServiceAgreement(ctx, entry.ServiceAgreementID)
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", err)
		}
		if !agreement.Active {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", ErrInactiveServiceAgreement)
		}
		if agreement.CustomerProfileID != cmd.CustomerProfileID {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: service agreement mismatch for %s", entry.ID)
		}
		if agreement.Currency != profile.DefaultCurrency {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: agreement currency %q must match customer currency %q", agreement.Currency, profile.DefaultCurrency)
		}

		rate, err := core.NewMoney(agreement.HourlyRate, agreement.Currency)
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: build rate snapshot: %w", err)
		}
		entryCopy := entry
		lockedEntries = append(lockedEntries, &entryCopy)
		line, err := core.NewInvoiceLine(core.InvoiceLineParams{
			InvoiceID:          "inv_seed",
			ServiceAgreementID: agreement.ID,
			TimeEntryID:        entry.ID,
			UnitRate:           rate,
		})
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: build invoice line: %w", err)
		}
		lines = append(lines, line)
	}

	invoice, err := core.NewInvoice(core.InvoiceParams{
		CustomerID: cmd.CustomerProfileID,
		Status:     core.InvoiceStatusDraft,
		Currency:   profile.DefaultCurrency,
		Lines:      lines,
	})
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", err)
	}

	for i := range lockedEntries {
		if err := lockedEntries[i].AssignToInvoice(invoice.ID); err != nil {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: assign entry: %w", err)
		}
		invoice.Lines[i].InvoiceID = invoice.ID
	}

	if err := s.invoices.CreateDraft(ctx, &invoice, lockedEntries); err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: save invoice: %w", err)
	}

	return invoiceToDTO(invoice, derefTimeEntries(lockedEntries)), nil
}

func (s InvoiceService) DiscardDraft(ctx context.Context, invoiceID string) error {
	_, err := s.Discard(ctx, invoiceID)
	return err
}

func (s InvoiceService) Discard(ctx context.Context, invoiceID string) (DiscardResult, error) {
	if s.invoices == nil || s.entries == nil {
		return DiscardResult{}, errors.New("invoice service dependencies are required")
	}

	invoice, err := s.getInvoice(ctx, invoiceID)
	if err != nil {
		return DiscardResult{}, fmt.Errorf("discard invoice: %w", err)
	}

	if invoice.IsDiscarded() {
		return DiscardResult{}, errors.New("discard invoice: invoice is already discarded")
	}

	if invoice.IsDraft() {
		// Hard-delete: the store Delete handles unlocking time entries atomically
		// with the invoice deletion in a single transaction.
		if err := s.invoices.Delete(ctx, invoiceID); err != nil {
			return DiscardResult{}, fmt.Errorf("discard invoice: delete invoice: %w", err)
		}
		return DiscardResult{WasSoftDiscard: false}, nil
	}

	// Issued: soft-discard via status transition.
	invoiceNumber := invoice.InvoiceNumber
	if err := invoice.Discard(time.Now().UTC()); err != nil {
		return DiscardResult{}, fmt.Errorf("discard invoice: %w", err)
	}
	if err := s.invoices.Update(ctx, invoice); err != nil {
		return DiscardResult{}, fmt.Errorf("discard invoice: update invoice: %w", err)
	}
	return DiscardResult{WasSoftDiscard: true, InvoiceNumber: invoiceNumber, Invoice: invoiceToDTO(*invoice, nil)}, nil
}

func (s InvoiceService) IssueDraft(ctx context.Context, cmd IssueInvoiceCommand) (InvoiceDTO, error) {
	if strings.TrimSpace(cmd.InvoiceID) == "" {
		return InvoiceDTO{}, errors.New("invoice id is required")
	}
	if s.invoices == nil || s.entries == nil || s.numbers == nil {
		return InvoiceDTO{}, errors.New("invoice service dependencies are required")
	}

	invoice, err := s.getInvoice(ctx, cmd.InvoiceID)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("issue invoice draft: %w", err)
	}
	if invoice == nil {
		return InvoiceDTO{}, errors.New("issue invoice draft: invoice is required")
	}
	if !invoice.IsDraft() {
		return InvoiceDTO{}, errors.New("issue invoice draft: invoice is not draft")
	}

	lockedEntries := make([]core.TimeEntry, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		entry, err := s.getTimeEntry(ctx, line.TimeEntryID)
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("issue invoice draft: %w", err)
		}
		if entry == nil {
			return InvoiceDTO{}, errors.New("issue invoice draft: time entry is required")
		}
		lockedEntries = append(lockedEntries, *entry)
	}

	number, err := s.numbers.Next(ctx)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("issue invoice draft: next invoice number: %w", err)
	}

	issuedAt := time.Now().UTC()
	if err := invoice.Issue(number, issuedAt); err != nil {
		return InvoiceDTO{}, fmt.Errorf("issue invoice draft: %w", err)
	}
	if err := s.invoices.Update(ctx, invoice); err != nil {
		return InvoiceDTO{}, fmt.Errorf("issue invoice draft: update invoice: %w", err)
	}

	return invoiceToDTO(*invoice, lockedEntries), nil
}

func derefTimeEntries(entries []*core.TimeEntry) []core.TimeEntry {
	res := make([]core.TimeEntry, 0, len(entries))
	for _, entry := range entries {
		if entry != nil {
			res = append(res, *entry)
		}
	}
	return res
}

func (s InvoiceService) getInvoice(ctx context.Context, id string) (*core.Invoice, error) {
	invoice, err := s.invoices.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrInvoiceNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	return invoice, nil
}

func (s InvoiceService) getTimeEntry(ctx context.Context, id string) (*core.TimeEntry, error) {
	entry, err := s.entries.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get time entry: %w", err)
	}
	return entry, nil
}

func (s InvoiceService) getServiceAgreement(ctx context.Context, id string) (*core.ServiceAgreement, error) {
	agreement, err := s.agreements.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrServiceAgreementNotFound) {
			return nil, ErrServiceAgreementNotFound
		}
		return nil, fmt.Errorf("get service agreement: %w", err)
	}
	return agreement, nil
}

func (s InvoiceService) getCustomerProfile(ctx context.Context, id string) (*core.CustomerProfile, error) {
	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return nil, ErrCustomerProfileNotFound
		}
		return nil, fmt.Errorf("get customer profile: %w", err)
	}
	return profile, nil
}
