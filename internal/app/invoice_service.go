package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrNoUnbilledEntries = errors.New("no unbilled time entries")
var ErrCustomerProfileInactive = errors.New("customer profile is inactive")

type InvoiceStore interface {
	CreateDraft(ctx context.Context, invoice *core.Invoice, entries []*core.TimeEntry) error
	GetByID(ctx context.Context, id string) (*core.Invoice, error)
	Delete(ctx context.Context, id string) error
}

type InvoiceService struct {
	invoices   InvoiceStore
	entries    TimeEntryStore
	agreements ServiceAgreementStore
	profiles   CustomerProfileStore
}

func NewInvoiceService(invoices InvoiceStore, entries TimeEntryStore, agreements ServiceAgreementStore, profiles CustomerProfileStore) InvoiceService {
	return InvoiceService{invoices: invoices, entries: entries, agreements: agreements, profiles: profiles}
}

func (s InvoiceService) CreateDraftFromUnbilled(ctx context.Context, cmd CreateDraftFromUnbilledCommand) (InvoiceDTO, error) {
	if strings.TrimSpace(cmd.CustomerProfileID) == "" {
		return InvoiceDTO{}, errors.New("customer profile id is required")
	}
	if s.profiles == nil || s.entries == nil || s.agreements == nil || s.invoices == nil {
		return InvoiceDTO{}, errors.New("invoice service dependencies are required")
	}

	profile, err := s.profiles.GetByID(ctx, cmd.CustomerProfileID)
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return InvoiceDTO{}, ErrCustomerProfileNotFound
		}
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: get customer profile: %w", err)
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
		agreement, err := s.agreements.GetByID(ctx, entry.ServiceAgreementID)
		if err != nil {
			if errors.Is(err, ErrServiceAgreementNotFound) {
				return InvoiceDTO{}, ErrServiceAgreementNotFound
			}
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: get service agreement: %w", err)
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
	if s.invoices == nil || s.entries == nil {
		return errors.New("invoice service dependencies are required")
	}

	invoice, err := s.invoices.GetByID(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("discard invoice draft: get invoice: %w", err)
	}
	if !invoice.IsDraft() {
		return errors.New("discard invoice draft: invoice is not draft")
	}

	for _, line := range invoice.Lines {
		entry, err := s.entries.GetByID(ctx, line.TimeEntryID)
		if err != nil {
			return fmt.Errorf("discard invoice draft: get time entry: %w", err)
		}
		entry.UnassignFromInvoice()
		if err := s.entries.Save(ctx, entry); err != nil {
			return fmt.Errorf("discard invoice draft: save time entry: %w", err)
		}
	}

	if err := s.invoices.Delete(ctx, invoiceID); err != nil {
		return fmt.Errorf("discard invoice draft: delete invoice: %w", err)
	}
	return nil
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
