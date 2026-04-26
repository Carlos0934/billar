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
var ErrInvalidStatusFilter = errors.New("invalid invoice status filter")

type InvoiceStore interface {
	CreateDraft(ctx context.Context, invoice *core.Invoice, entries []*core.TimeEntry) error
	GetByID(ctx context.Context, id string) (*core.Invoice, error)
	Update(ctx context.Context, invoice *core.Invoice) error
	Delete(ctx context.Context, id string) error
	ListByCustomer(ctx context.Context, customerID string, status ...core.InvoiceStatus) ([]core.InvoiceSummary, error)
	AddLine(ctx context.Context, invoiceID string, line core.InvoiceLine) error
	RemoveLine(ctx context.Context, invoiceID, lineID string) error
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
	issuers    DefaultIssuerProfileStore
}

func NewInvoiceService(invoices InvoiceStore, entries TimeEntryStore, agreements ServiceAgreementStore, profiles CustomerProfileStore, optional ...any) InvoiceService {
	var numberGen InvoiceNumberGenerator
	var issuers DefaultIssuerProfileStore
	for _, opt := range optional {
		switch v := opt.(type) {
		case InvoiceNumberGenerator:
			numberGen = v
		case DefaultIssuerProfileStore:
			issuers = v
		}
	}
	return InvoiceService{invoices: invoices, entries: entries, agreements: agreements, profiles: profiles, numbers: numberGen, issuers: issuers}
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
	billableEntries := entries[:0]
	for _, entry := range entries {
		if entry.Billable {
			billableEntries = append(billableEntries, entry)
		}
	}
	if len(billableEntries) == 0 {
		return InvoiceDTO{}, ErrNoUnbilledEntries
	}
	entries = billableEntries

	periodStart, err := parseInvoiceCommandDate(cmd.PeriodStart, "period_start")
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", err)
	}
	periodEnd, err := parseInvoiceCommandDate(cmd.PeriodEnd, "period_end")
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", err)
	}
	dueDate, err := parseInvoiceCommandDate(cmd.DueDate, "due_date")
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("create invoice draft: %w", err)
	}
	if periodStart.IsZero() || periodEnd.IsZero() {
		defaultStart, defaultEnd := invoicePeriodFromEntries(entries)
		if periodStart.IsZero() {
			periodStart = defaultStart
		}
		if periodEnd.IsZero() {
			periodEnd = defaultEnd
		}
	}
	notes := cmd.Notes
	if notes == "" && s.issuers != nil {
		issuer, err := s.issuers.GetDefault(ctx)
		if err != nil && !errors.Is(err, ErrIssuerProfileNotFound) {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: get issuer default notes: %w", err)
		}
		if issuer != nil {
			notes = issuer.DefaultNotes
		}
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
			Description:        entry.Description,
			QuantityMin:        int64(entry.Hours) * 60 / 10000,
			UnitRate:           rate,
		})
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("create invoice draft: build invoice line: %w", err)
		}
		lines = append(lines, line)
	}

	invoice, err := core.NewInvoice(core.InvoiceParams{
		CustomerID:  cmd.CustomerProfileID,
		Status:      core.InvoiceStatusDraft,
		Currency:    profile.DefaultCurrency,
		Lines:       lines,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		DueDate:     dueDate,
		Notes:       notes,
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

func (s InvoiceService) GetInvoice(ctx context.Context, id string) (InvoiceDTO, error) {
	if strings.TrimSpace(id) == "" {
		return InvoiceDTO{}, errors.New("invoice id is required")
	}
	inv, err := s.getInvoice(ctx, id)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("get invoice: %w", err)
	}

	entries := make([]core.TimeEntry, 0, len(inv.Lines))
	for _, line := range inv.Lines {
		if strings.TrimSpace(line.TimeEntryID) == "" {
			continue
		}
		entry, err := s.getTimeEntry(ctx, line.TimeEntryID)
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("get invoice: load time entry: %w", err)
		}
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return invoiceToDTO(*inv, entries), nil
}

func (s InvoiceService) AddDraftLine(ctx context.Context, cmd AddDraftLineCommand) (InvoiceDTO, error) {
	if strings.TrimSpace(cmd.InvoiceID) == "" {
		return InvoiceDTO{}, errors.New("invoice id is required")
	}
	if s.invoices == nil {
		return InvoiceDTO{}, errors.New("invoice service dependencies are required")
	}
	invoice, err := s.getInvoice(ctx, cmd.InvoiceID)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("add draft invoice line: %w", err)
	}
	if invoice == nil {
		return InvoiceDTO{}, errors.New("add draft invoice line: invoice is required")
	}
	if !invoice.IsDraft() {
		return InvoiceDTO{}, errors.New("add draft invoice line: invoice is not draft")
	}
	if strings.TrimSpace(cmd.Description) == "" {
		return InvoiceDTO{}, errors.New("add draft invoice line: invoice line description is required")
	}
	rate, err := core.NewMoney(cmd.UnitRate, strings.TrimSpace(cmd.Currency))
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("add draft invoice line: %w", err)
	}
	serviceAgreementID := "manual"
	if len(invoice.Lines) > 0 && invoice.Lines[0].ServiceAgreementID != "" {
		serviceAgreementID = invoice.Lines[0].ServiceAgreementID
	}
	line, err := core.NewManualInvoiceLine(invoice.ID, serviceAgreementID, cmd.Description, cmd.QuantityMin, rate, invoice.Currency)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("add draft invoice line: %w", err)
	}
	if err := invoice.AddLine(line); err != nil {
		return InvoiceDTO{}, fmt.Errorf("add draft invoice line: %w", err)
	}
	if err := s.invoices.AddLine(ctx, invoice.ID, line); err != nil {
		return InvoiceDTO{}, fmt.Errorf("add draft invoice line: save line: %w", err)
	}
	return invoiceToDTO(*invoice, nil), nil
}

func (s InvoiceService) RemoveDraftLine(ctx context.Context, cmd RemoveDraftLineCommand) (InvoiceDTO, error) {
	if strings.TrimSpace(cmd.InvoiceID) == "" {
		return InvoiceDTO{}, errors.New("invoice id is required")
	}
	if strings.TrimSpace(cmd.InvoiceLineID) == "" {
		return InvoiceDTO{}, errors.New("invoice line id is required")
	}
	if s.invoices == nil {
		return InvoiceDTO{}, errors.New("invoice service dependencies are required")
	}
	invoice, err := s.getInvoice(ctx, cmd.InvoiceID)
	if err != nil {
		return InvoiceDTO{}, fmt.Errorf("remove draft invoice line: %w", err)
	}
	if invoice == nil {
		return InvoiceDTO{}, errors.New("remove draft invoice line: invoice is required")
	}
	if !invoice.IsDraft() {
		return InvoiceDTO{}, errors.New("remove draft invoice line: invoice is not draft")
	}
	if err := invoice.RemoveLine(cmd.InvoiceLineID); err != nil {
		return InvoiceDTO{}, fmt.Errorf("remove draft invoice line: %w", err)
	}
	if err := s.invoices.RemoveLine(ctx, invoice.ID, strings.TrimSpace(cmd.InvoiceLineID)); err != nil {
		return InvoiceDTO{}, fmt.Errorf("remove draft invoice line: delete line: %w", err)
	}
	return invoiceToDTO(*invoice, nil), nil
}

func (s InvoiceService) ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]InvoiceSummaryDTO, error) {
	if strings.TrimSpace(customerID) == "" {
		return nil, errors.New("customer id is required")
	}

	var statuses []core.InvoiceStatus
	if strings.TrimSpace(statusFilter) != "" {
		s := core.InvoiceStatus(statusFilter)
		if !s.IsValid() {
			return nil, fmt.Errorf("list invoices: %w", ErrInvalidStatusFilter)
		}
		statuses = append(statuses, s)
	}

	summaries, err := s.invoices.ListByCustomer(ctx, customerID, statuses...)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	dtos := make([]InvoiceSummaryDTO, 0, len(summaries))
	for _, s := range summaries {
		dtos = append(dtos, InvoiceSummaryDTO{
			ID:            s.ID,
			InvoiceNumber: s.InvoiceNumber,
			CustomerID:    s.CustomerID,
			Status:        string(s.Status),
			Currency:      s.Currency,
			PeriodStart:   formatInvoiceTime(s.PeriodStart),
			PeriodEnd:     formatInvoiceTime(s.PeriodEnd),
			DueDate:       formatInvoiceTime(s.DueDate),
			GrandTotal:    s.GrandTotal,
			CreatedAt:     formatInvoiceTime(s.CreatedAt),
		})
	}
	return dtos, nil
}

func parseInvoiceCommandDate(value, field string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be RFC3339 or YYYY-MM-DD", field)
	}
	return t.UTC(), nil
}

func invoicePeriodFromEntries(entries []core.TimeEntry) (time.Time, time.Time) {
	var start, end time.Time
	for _, entry := range entries {
		date := entry.Date.UTC()
		if start.IsZero() || date.Before(start) {
			start = date
		}
		if end.IsZero() || date.After(end) {
			end = date
		}
	}
	return start, end
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
		if strings.TrimSpace(line.TimeEntryID) == "" {
			continue
		}
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
