package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type invoiceStoreStub struct {
	createDraftInvoice         *core.Invoice
	createDraftEntries         []*core.TimeEntry
	updateInvoice              *core.Invoice
	updateErr                  error
	createDraftErr             error
	getByIDRes                 *core.Invoice
	getByIDErr                 error
	deleteID                   string
	deleteErr                  error
	listByCustomerRes          []core.InvoiceSummary
	listByCustomerErr          error
	listByCustomerStatusFilter string
	addLineInvoiceID           string
	addLineLine                core.InvoiceLine
	addLineErr                 error
	removeLineInvoiceID        string
	removeLineID               string
	removeLineErr              error
}

func (s *invoiceStoreStub) CreateDraft(ctx context.Context, invoice *core.Invoice, entries []*core.TimeEntry) error {
	_ = ctx
	s.createDraftInvoice = invoice
	s.createDraftEntries = entries
	return s.createDraftErr
}

func (s *invoiceStoreStub) GetByID(ctx context.Context, id string) (*core.Invoice, error) {
	_ = ctx
	return s.getByIDRes, s.getByIDErr
}

func (s *invoiceStoreStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func (s *invoiceStoreStub) Update(ctx context.Context, invoice *core.Invoice) error {
	_ = ctx
	s.updateInvoice = invoice
	return s.updateErr
}

func (s *invoiceStoreStub) ListByCustomer(ctx context.Context, customerID string, status ...core.InvoiceStatus) ([]core.InvoiceSummary, error) {
	_ = ctx
	if s.listByCustomerStatusFilter != "" && (len(status) == 0 || string(status[0]) != s.listByCustomerStatusFilter) {
		return nil, nil
	}
	return s.listByCustomerRes, s.listByCustomerErr
}

func (s *invoiceStoreStub) AddLine(ctx context.Context, invoiceID string, line core.InvoiceLine) error {
	_ = ctx
	s.addLineInvoiceID = invoiceID
	s.addLineLine = line
	return s.addLineErr
}

func (s *invoiceStoreStub) RemoveLine(ctx context.Context, invoiceID, lineID string) error {
	_ = ctx
	s.removeLineInvoiceID = invoiceID
	s.removeLineID = lineID
	return s.removeLineErr
}

type invoiceNumberGeneratorStub struct {
	next      string
	err       error
	callCount int
}

type defaultIssuerProfileStoreStub struct {
	profile *core.IssuerProfile
	err     error
}

func (s *defaultIssuerProfileStoreStub) Save(context.Context, *core.IssuerProfile) error { return nil }
func (s *defaultIssuerProfileStoreStub) GetByID(context.Context, string) (*core.IssuerProfile, error) {
	return s.profile, s.err
}
func (s *defaultIssuerProfileStoreStub) Delete(context.Context, string) error { return nil }
func (s *defaultIssuerProfileStoreStub) GetDefault(context.Context) (*core.IssuerProfile, error) {
	return s.profile, s.err
}

func (s *invoiceNumberGeneratorStub) Next(ctx context.Context) (string, error) {
	_ = ctx
	s.callCount++
	return s.next, s.err
}

func makeInvoiceServiceFixtures() (*customerProfileStoreForTimeEntry, *serviceAgreementStoreForTimeEntry, *timeEntryStoreStub, *invoiceStoreStub) {
	return &customerProfileStoreForTimeEntry{getByIDRes: activeProfile()}, &serviceAgreementStoreForTimeEntry{getByIDRes: activeAgreement()}, &timeEntryStoreStub{}, &invoiceStoreStub{}
}

func TestInvoiceServiceCreateDraftFromUnbilled_HappyPath(t *testing.T) {
	t.Parallel()

	profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
	hours1, _ := core.NewHours(15000)
	hours2, _ := core.NewHours(30000)
	entries.listUnbilledRes = []core.TimeEntry{
		{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work A", Hours: hours1, Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)},
		{ID: "te_002", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work B", Hours: hours2, Billable: true, Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)},
	}

	svc := NewInvoiceService(invoices, entries, agreements, profiles)
	dto, err := svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if err != nil {
		t.Fatalf("CreateDraftFromUnbilled() error = %v", err)
	}
	if invoices.createDraftInvoice == nil {
		t.Fatal("CreateDraft was not called")
	}
	if len(invoices.createDraftEntries) != 2 {
		t.Fatalf("len(createDraftEntries) = %d, want 2", len(invoices.createDraftEntries))
	}
	if !dto.IsDraft {
		t.Fatal("returned invoice DTO should be draft")
	}
	if dto.CustomerID != "cus_abc123" {
		t.Fatalf("CustomerID = %q, want cus_abc123", dto.CustomerID)
	}
	if len(dto.Lines) != 2 {
		t.Fatalf("len(Lines) = %d, want 2", len(dto.Lines))
	}
	if dto.Lines[0].Description != "Work A" || dto.Lines[1].Description != "Work B" {
		t.Fatalf("line descriptions = %#v, want derived from time entries", dto.Lines)
	}
	if dto.Lines[0].QuantityMin != 90 || dto.Lines[1].QuantityMin != 180 {
		t.Fatalf("quantity mins = %d, %d, want 90, 180", dto.Lines[0].QuantityMin, dto.Lines[1].QuantityMin)
	}
	if dto.Lines[0].LineTotalAmount != 1500 || dto.Lines[1].LineTotalAmount != 3000 {
		t.Fatalf("line totals = %d, %d, want 1500, 3000", dto.Lines[0].LineTotalAmount, dto.Lines[1].LineTotalAmount)
	}
	if len(invoices.createDraftEntries) > 0 {
		if err := invoices.createDraftEntries[0].Update("should fail", hours1); !errors.Is(err, core.ErrTimeEntryLocked) {
			t.Fatalf("locked entry Update() error = %v, want ErrTimeEntryLocked", err)
		}
	}
}

func TestInvoiceServiceCreateDraftFromUnbilled_Metadata(t *testing.T) {
	t.Parallel()

	issuer, _ := core.NewIssuerProfile(core.IssuerProfileParams{LegalEntityID: "le_issuer", DefaultCurrency: "USD", DefaultNotes: "Net 15"})
	tests := []struct {
		name      string
		cmd       CreateDraftFromUnbilledCommand
		issuer    *core.IssuerProfile
		wantStart string
		wantEnd   string
		wantDue   string
		wantNotes string
		wantErr   string
	}{
		{name: "explicit metadata wins", cmd: CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123", PeriodStart: "2026-04-01", PeriodEnd: "2026-04-30", DueDate: "2026-05-15", Notes: "Custom terms"}, issuer: &issuer, wantStart: "2026-04-01T00:00:00Z", wantEnd: "2026-04-30T00:00:00Z", wantDue: "2026-05-15T00:00:00Z", wantNotes: "Custom terms"},
		{name: "period defaults to min max and notes from issuer", cmd: CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"}, issuer: &issuer, wantStart: "2026-04-02T00:00:00Z", wantEnd: "2026-04-18T00:00:00Z", wantNotes: "Net 15"},
		{name: "missing issuer defaults notes empty", cmd: CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"}, issuer: nil, wantStart: "2026-04-02T00:00:00Z", wantEnd: "2026-04-18T00:00:00Z", wantNotes: ""},
		{name: "reject invalid period", cmd: CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123", PeriodStart: "2026-05-01", PeriodEnd: "2026-04-30"}, issuer: &issuer, wantErr: "period_end"},
		{name: "reject due before period", cmd: CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123", PeriodEnd: "2026-04-30", DueDate: "2026-04-15"}, issuer: &issuer, wantErr: "due_date"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
			entries.listUnbilledRes = []core.TimeEntry{
				{ID: "te_early", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Early", Hours: mustHours(10000), Billable: true, Date: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)},
				{ID: "te_late", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Late", Hours: mustHours(10000), Billable: true, Date: time.Date(2026, 4, 18, 0, 0, 0, 0, time.UTC)},
			}
			svc := NewInvoiceService(invoices, entries, agreements, profiles, &defaultIssuerProfileStoreStub{profile: tc.issuer})
			dto, err := svc.CreateDraftFromUnbilled(context.Background(), tc.cmd)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("CreateDraftFromUnbilled() error = %v, want %q", err, tc.wantErr)
				}
				if invoices.createDraftInvoice != nil || len(invoices.createDraftEntries) != 0 {
					t.Fatalf("invalid metadata wrote invoice=%#v entries=%d", invoices.createDraftInvoice, len(invoices.createDraftEntries))
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateDraftFromUnbilled() error = %v", err)
			}
			if dto.PeriodStart != tc.wantStart || dto.PeriodEnd != tc.wantEnd || dto.DueDate != tc.wantDue || dto.Notes != tc.wantNotes {
				t.Fatalf("dto metadata = (%q,%q,%q,%q), want (%q,%q,%q,%q)", dto.PeriodStart, dto.PeriodEnd, dto.DueDate, dto.Notes, tc.wantStart, tc.wantEnd, tc.wantDue, tc.wantNotes)
			}
			if invoices.createDraftInvoice == nil || invoices.createDraftInvoice.Notes != tc.wantNotes {
				t.Fatalf("stored invoice metadata = %#v", invoices.createDraftInvoice)
			}
		})
	}
}

func TestInvoiceServiceCreateDraftFromUnbilled_SkipsNonBillableEntries(t *testing.T) {
	t.Parallel()

	profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
	entries.listUnbilledRes = []core.TimeEntry{
		{ID: "te_billable_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Billable work", Hours: mustHours(9000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)},
		{ID: "te_nonbillable_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Internal admin", Hours: mustHours(6000), Billable: false, Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)},
	}

	svc := NewInvoiceService(invoices, entries, agreements, profiles)
	dto, err := svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if err != nil {
		t.Fatalf("CreateDraftFromUnbilled() error = %v", err)
	}
	if len(dto.Lines) != 1 {
		t.Fatalf("len(Lines) = %d, want 1 billable line", len(dto.Lines))
	}
	if dto.Lines[0].TimeEntryID != "te_billable_001" {
		t.Fatalf("Lines[0].TimeEntryID = %q, want te_billable_001", dto.Lines[0].TimeEntryID)
	}
	if dto.Lines[0].Description != "Billable work" {
		t.Fatalf("Lines[0].Description = %q, want Billable work", dto.Lines[0].Description)
	}
	if len(invoices.createDraftEntries) != 1 {
		t.Fatalf("len(createDraftEntries) = %d, want 1 locked billable entry", len(invoices.createDraftEntries))
	}
	if invoices.createDraftEntries[0].ID != "te_billable_001" {
		t.Fatalf("locked entry ID = %q, want te_billable_001", invoices.createDraftEntries[0].ID)
	}
	if !invoices.createDraftEntries[0].Billable {
		t.Fatal("locked entry is non-billable, want billable only")
	}
}

func TestInvoiceServiceCreateDraftFromUnbilled_ReturnsNoUnbilledWhenOnlyNonBillable(t *testing.T) {
	t.Parallel()

	profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
	agreements.getByIDErr = errors.New("agreement lookup should not run for non-billable entries")
	entries.listUnbilledRes = []core.TimeEntry{
		{ID: "te_nonbillable_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Internal admin", Hours: mustHours(6000), Billable: false, Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)},
		{ID: "te_nonbillable_002", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Sales support", Hours: mustHours(3000), Billable: false, Date: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)},
	}

	svc := NewInvoiceService(invoices, entries, agreements, profiles)
	_, err := svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if !errors.Is(err, ErrNoUnbilledEntries) {
		t.Fatalf("CreateDraftFromUnbilled() error = %v, want ErrNoUnbilledEntries", err)
	}
	if invoices.createDraftInvoice != nil {
		t.Fatalf("CreateDraft invoice = %#v, want no invoice write", invoices.createDraftInvoice)
	}
	if len(invoices.createDraftEntries) != 0 {
		t.Fatalf("len(createDraftEntries) = %d, want 0 locked entries", len(invoices.createDraftEntries))
	}
}

func TestInvoiceServiceCreateDraftFromUnbilled_Rejections(t *testing.T) {
	t.Parallel()

	profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
	svc := NewInvoiceService(invoices, entries, agreements, profiles)

	_, err := svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if !errors.Is(err, ErrNoUnbilledEntries) {
		t.Fatalf("empty unbilled error = %v, want ErrNoUnbilledEntries", err)
	}

	entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work A", Hours: func() core.Hours { h, _ := core.NewHours(15000); return h }(), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
	profiles.getByIDErr = ErrCustomerProfileNotFound
	_, err = svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_missing"})
	if !errors.Is(err, ErrCustomerProfileNotFound) {
		t.Fatalf("missing customer error = %v, want ErrCustomerProfileNotFound", err)
	}

	profiles.getByIDErr = nil
	profiles.getByIDRes = activeProfile()
	agreements.getByIDErr = ErrServiceAgreementNotFound
	_, err = svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if !errors.Is(err, ErrServiceAgreementNotFound) {
		t.Fatalf("missing agreement error = %v, want ErrServiceAgreementNotFound", err)
	}

	profiles.getByIDRes = nil
	profiles.getByIDErr = nil
	agreements.getByIDErr = nil
	agreements.getByIDRes = activeAgreement()
	_, err = svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if err == nil || !strings.Contains(err.Error(), "customer profile is required") {
		t.Fatalf("nil profile error = %v, want customer profile is required", err)
	}

	profiles.getByIDRes = inactiveProfile()
	profiles.getByIDErr = nil
	agreements.getByIDErr = nil
	agreements.getByIDRes = inactiveAgreement()
	_, err = svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if !errors.Is(err, ErrCustomerProfileInactive) {
		t.Fatalf("inactive customer error = %v, want ErrCustomerProfileInactive", err)
	}
}

func TestInvoiceServiceCreateDraftFromUnbilled_DependencyAndStoreFailures(t *testing.T) {
	t.Parallel()

	profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
	entries.listUnbilledErr = errors.New("list failed")
	svc := NewInvoiceService(invoices, entries, agreements, profiles)

	_, err := svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if err == nil || !strings.Contains(err.Error(), "list failed") {
		t.Fatalf("ListUnbilled error = %v, want propagated list failure", err)
	}

	entries.listUnbilledErr = nil
	entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work A", Hours: func() core.Hours { h, _ := core.NewHours(15000); return h }(), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
	invoices.createDraftErr = errors.New("save failed")
	_, err = svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"})
	if err == nil || !strings.Contains(err.Error(), "save failed") {
		t.Fatalf("CreateDraft save error = %v, want propagated save failure", err)
	}
}

func TestInvoiceServiceDiscardDraft(t *testing.T) {
	t.Parallel()

	entry1, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work A", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)})
	entry2, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work B", Hours: mustHours(30000), Billable: true, Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)})
	_ = entry1.AssignToInvoice("inv_001")
	_ = entry2.AssignToInvoice("inv_001")
	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry1.ID, UnitRate: rate})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry2.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1, line2}})
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, agreementsForInvoice(), profilesForInvoice())

	if err := svc.DiscardDraft(context.Background(), invoice.ID); err != nil {
		t.Fatalf("DiscardDraft() error = %v", err)
	}
	if invoices.deleteID != invoice.ID {
		t.Fatalf("Delete called with %q, want %q", invoices.deleteID, invoice.ID)
	}
	// Entry unlocking is handled atomically by the store layer (integration-tested).
}

func TestInvoiceServiceDiscardDraft_SoftDiscardsIssuedInvoice(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	entry, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: hours, Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)})
	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusIssued, Currency: "USD", Lines: []core.InvoiceLine{line}})
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{getByIDRes: &entry}, agreementsForInvoice(), profilesForInvoice())

	if err := svc.DiscardDraft(context.Background(), invoice.ID); err != nil {
		t.Fatalf("DiscardDraft() error = %v, want soft-discard success", err)
	}
	if invoices.updateInvoice == nil {
		t.Fatal("Update was not called for soft-discard")
	}
	if invoices.updateInvoice.Status != core.InvoiceStatusDiscarded {
		t.Fatalf("updated status = %q, want discarded", invoices.updateInvoice.Status)
	}
}

func TestInvoiceServiceIssueDraft(t *testing.T) {
	t.Parallel()

	hours := mustHours(15000)
	entry := mustIssueDraftEntry("te_001", "Work A", hours)
	svc, invoices, _ := newIssueDraftService(t, invoiceWithSingleLine("inv_001", entry.ID, core.InvoiceStatusDraft), entry, &invoiceNumberGeneratorStub{next: "INV-2026-0001"})
	dto, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: "inv_001"})
	if err != nil {
		t.Fatalf("IssueDraft() error = %v", err)
	}
	if dto.Status != string(core.InvoiceStatusIssued) {
		t.Fatalf("Status = %q, want issued", dto.Status)
	}
	if dto.IsDraft {
		t.Fatal("IsDraft should be false after issue")
	}
	if !dto.IsIssued {
		t.Fatal("IsIssued should be true after issue")
	}
	if dto.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", dto.InvoiceNumber)
	}
	if dto.IssuedAt == "" {
		t.Fatal("IssuedAt should be set")
	}
	if invoices.updateInvoice == nil {
		t.Fatal("Update was not called")
	}
	if invoices.updateInvoice.Status != core.InvoiceStatusIssued {
		t.Fatalf("stored status = %q, want issued", invoices.updateInvoice.Status)
	}
}

func TestInvoiceServiceIssueDraft_PreservesMetadataFromCreatedDraft(t *testing.T) {
	t.Parallel()

	profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
	entry := core.TimeEntry{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Metadata work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 12, 0, 0, 0, 0, time.UTC)}
	entries.listUnbilledRes = []core.TimeEntry{entry}
	entries.getByIDRes = &entry
	issuer, _ := core.NewIssuerProfile(core.IssuerProfileParams{LegalEntityID: "le_issuer", DefaultCurrency: "USD", DefaultNotes: "Issuer default must not replace explicit notes"})
	svc := NewInvoiceService(invoices, entries, agreements, profiles, &invoiceNumberGeneratorStub{next: "INV-2026-0001"}, &defaultIssuerProfileStoreStub{profile: &issuer})

	draft, err := svc.CreateDraftFromUnbilled(context.Background(), CreateDraftFromUnbilledCommand{
		CustomerProfileID: "cus_abc123",
		PeriodStart:       "2026-04-01",
		PeriodEnd:         "2026-04-30",
		DueDate:           "2026-05-15",
		Notes:             "Keep these billing notes",
	})
	if err != nil {
		t.Fatalf("CreateDraftFromUnbilled() error = %v", err)
	}
	if invoices.createDraftInvoice == nil {
		t.Fatal("CreateDraft was not called")
	}
	beforeStart, beforeEnd, beforeDue, beforeNotes := draft.PeriodStart, draft.PeriodEnd, draft.DueDate, draft.Notes

	invoices.getByIDRes = invoices.createDraftInvoice
	issued, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: invoices.createDraftInvoice.ID})
	if err != nil {
		t.Fatalf("IssueDraft() error = %v", err)
	}

	if issued.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", issued.InvoiceNumber)
	}
	if issued.PeriodStart != beforeStart || issued.PeriodEnd != beforeEnd || issued.DueDate != beforeDue || issued.Notes != beforeNotes {
		t.Fatalf("issued metadata = (%q,%q,%q,%q), want unchanged (%q,%q,%q,%q)", issued.PeriodStart, issued.PeriodEnd, issued.DueDate, issued.Notes, beforeStart, beforeEnd, beforeDue, beforeNotes)
	}
	if invoices.updateInvoice == nil {
		t.Fatal("Update was not called")
	}
	if gotStart, gotEnd, gotDue, gotNotes := formatInvoiceTime(invoices.updateInvoice.PeriodStart), formatInvoiceTime(invoices.updateInvoice.PeriodEnd), formatInvoiceTime(invoices.updateInvoice.DueDate), invoices.updateInvoice.Notes; gotStart != beforeStart || gotEnd != beforeEnd || gotDue != beforeDue || gotNotes != beforeNotes {
		t.Fatalf("stored issued metadata = (%q,%q,%q,%q), want unchanged (%q,%q,%q,%q)", gotStart, gotEnd, gotDue, gotNotes, beforeStart, beforeEnd, beforeDue, beforeNotes)
	}
}

func TestInvoiceServiceIssueDraft_Rejections(t *testing.T) {
	t.Parallel()

	svc, invoices, _ := newIssueDraftService(t, nil, nil, &invoiceNumberGeneratorStub{next: "INV-2026-0001"})

	if _, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{}); err == nil {
		t.Fatal("IssueDraft() error = nil, want blank invoice id rejected")
	}

	invoices.getByIDRes = invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusIssued)
	if _, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: "inv_001"}); err == nil {
		t.Fatal("IssueDraft() error = nil, want issued invoice rejected")
	}
}

func TestInvoiceServiceIssueDraft_ReissueDoesNotInvokeGenerator(t *testing.T) {
	t.Parallel()

	numbers := &invoiceNumberGeneratorStub{next: "INV-2026-0001"}
	invoice := invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusIssued)
	invoices := &invoiceStoreStub{getByIDRes: invoice}
	entries := &timeEntryStoreStub{getByIDRes: mustIssueDraftEntry("te_001", "Work", mustHours(15000))}
	svc := NewInvoiceService(invoices, entries, nil, nil, numbers)

	_, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: "inv_001"})
	if err == nil {
		t.Fatal("IssueDraft() error = nil, want already-issued rejection")
	}
	if numbers.callCount != 0 {
		t.Fatalf("generator invoked %d times on re-issue, want 0", numbers.callCount)
	}
}

func TestInvoiceServiceIssueDraft_NumberGeneratorFailure(t *testing.T) {
	t.Parallel()

	entry := mustIssueDraftEntry("te_001", "Work A", mustHours(15000))
	svc, _, _ := newIssueDraftService(t, invoiceWithSingleLine("inv_001", entry.ID, core.InvoiceStatusDraft), entry, &invoiceNumberGeneratorStub{err: errors.New("number failed")})
	if _, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: "inv_001"}); err == nil || !strings.Contains(err.Error(), "number failed") {
		t.Fatalf("IssueDraft() error = %v, want number failure", err)
	}
}

func TestInvoiceServiceIssueDraft_StoreUpdateFailure(t *testing.T) {
	t.Parallel()

	entry := mustIssueDraftEntry("te_001", "Work A", mustHours(15000))
	svc, invoices, _ := newIssueDraftService(t, invoiceWithSingleLine("inv_001", entry.ID, core.InvoiceStatusDraft), entry, &invoiceNumberGeneratorStub{next: "INV-2026-0001"})
	invoices.updateErr = errors.New("update failed")
	_, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: "inv_001"})
	if err == nil || !strings.Contains(err.Error(), "update failed") {
		t.Fatalf("IssueDraft() error = %v, want update failure", err)
	}
	if invoices.updateInvoice == nil || invoices.updateInvoice.Status != core.InvoiceStatusIssued {
		t.Fatalf("update invoice state = %#v, want issued invoice before update failure", invoices.updateInvoice)
	}
}

func TestInvoiceServiceAddDraftLine(t *testing.T) {
	t.Parallel()

	rate, _ := core.NewMoney(10000, "USD")
	base, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: "te_001", UnitRate: rate, Description: "Base work", QuantityMin: 60})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{base}})
	invoice.ID = "inv_001"
	invoice.Lines[0].InvoiceID = invoice.ID
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, nil, nil)

	dto, err := svc.AddDraftLine(context.Background(), AddDraftLineCommand{InvoiceID: "inv_001", Description: "Setup fee", QuantityMin: 60, UnitRate: 5000, Currency: "USD"})
	if err != nil {
		t.Fatalf("AddDraftLine() error = %v", err)
	}
	if invoices.addLineInvoiceID != "inv_001" {
		t.Fatalf("AddLine invoice id = %q, want inv_001", invoices.addLineInvoiceID)
	}
	if invoices.addLineLine.TimeEntryID != "" || invoices.addLineLine.Description != "Setup fee" || invoices.addLineLine.QuantityMin != 60 || invoices.addLineLine.UnitRate.Amount != 5000 {
		t.Fatalf("added line = %+v, want manual snapshot", invoices.addLineLine)
	}
	if len(dto.Lines) != 2 || dto.GrandTotal != 15000 {
		t.Fatalf("dto lines/total = %d/%d, want 2/15000", len(dto.Lines), dto.GrandTotal)
	}
	if dto.Lines[1].TimeEntryID != "" || dto.Lines[1].Description != "Setup fee" || dto.Lines[1].LineTotalAmount != 5000 {
		t.Fatalf("manual dto line = %+v", dto.Lines[1])
	}
}

func TestInvoiceServiceAddDraftLineRejections(t *testing.T) {
	t.Parallel()

	base := invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusDraft)
	tests := []struct {
		name    string
		invoice *core.Invoice
		cmd     AddDraftLineCommand
		want    string
	}{
		{name: "issued immutable", invoice: invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusIssued), cmd: AddDraftLineCommand{InvoiceID: "inv_001", Description: "Setup", QuantityMin: 60, UnitRate: 5000, Currency: "USD"}, want: "not draft"},
		{name: "discarded immutable", invoice: invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusDiscarded), cmd: AddDraftLineCommand{InvoiceID: "inv_001", Description: "Setup", QuantityMin: 60, UnitRate: 5000, Currency: "USD"}, want: "not draft"},
		{name: "currency mismatch", invoice: base, cmd: AddDraftLineCommand{InvoiceID: "inv_001", Description: "Setup", QuantityMin: 60, UnitRate: 5000, Currency: "EUR"}, want: "currency"},
		{name: "bad input", invoice: base, cmd: AddDraftLineCommand{InvoiceID: "inv_001", Description: " ", QuantityMin: 0, UnitRate: 0, Currency: "USD"}, want: "description"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			invoices := &invoiceStoreStub{getByIDRes: tc.invoice}
			svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, nil, nil)
			_, err := svc.AddDraftLine(context.Background(), tc.cmd)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("AddDraftLine() error = %v, want %q", err, tc.want)
			}
			if invoices.addLineInvoiceID != "" {
				t.Fatalf("AddLine called for rejected command: %+v", invoices.addLineLine)
			}
		})
	}
}

func TestInvoiceServiceRemoveDraftLine(t *testing.T) {
	t.Parallel()

	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: "te_001", UnitRate: rate, Description: "Base work", QuantityMin: 60})
	line2, _ := core.NewManualInvoiceLine("inv_seed", "sa_xyz789", "Setup fee", 60, core.Money{Amount: 5000, Currency: "USD"}, "USD")
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1, line2}})
	invoice.ID = "inv_001"
	for i := range invoice.Lines {
		invoice.Lines[i].InvoiceID = invoice.ID
	}
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, nil, nil)

	dto, err := svc.RemoveDraftLine(context.Background(), RemoveDraftLineCommand{InvoiceID: "inv_001", InvoiceLineID: line2.ID})
	if err != nil {
		t.Fatalf("RemoveDraftLine() error = %v", err)
	}
	if invoices.removeLineInvoiceID != "inv_001" || invoices.removeLineID != line2.ID {
		t.Fatalf("RemoveLine args = %q/%q, want inv_001/%s", invoices.removeLineInvoiceID, invoices.removeLineID, line2.ID)
	}
	if len(dto.Lines) != 1 || dto.Lines[0].ID != line1.ID || dto.GrandTotal != 10000 {
		t.Fatalf("dto after remove = %+v, want only line1 total 10000", dto)
	}
}

func TestInvoiceServiceRemoveDraftLineRejections(t *testing.T) {
	t.Parallel()

	draft := invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusDraft)
	tests := []struct {
		name    string
		invoice *core.Invoice
		lineID  string
		want    string
	}{
		{name: "last line", invoice: draft, lineID: draft.Lines[0].ID, want: "last"},
		{name: "unknown line", invoice: draft, lineID: "inl_missing", want: "not found"},
		{name: "issued immutable", invoice: invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusIssued), lineID: "anything", want: "not draft"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			invoices := &invoiceStoreStub{getByIDRes: tc.invoice}
			svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, nil, nil)
			_, err := svc.RemoveDraftLine(context.Background(), RemoveDraftLineCommand{InvoiceID: "inv_001", InvoiceLineID: tc.lineID})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("RemoveDraftLine() error = %v, want %q", err, tc.want)
			}
			if invoices.removeLineInvoiceID != "" {
				t.Fatalf("RemoveLine called for rejected command: %q/%q", invoices.removeLineInvoiceID, invoices.removeLineID)
			}
		})
	}
}

func TestInvoiceServiceIssueDraft_GeneratorFailureLeavesDraftUnpersisted(t *testing.T) {
	t.Parallel()

	entry := mustIssueDraftEntry("te_001", "Work A", mustHours(15000))
	invoice := invoiceWithSingleLine("inv_001", entry.ID, core.InvoiceStatusDraft)
	svc, invoices, _ := newIssueDraftService(t, invoice, entry, &invoiceNumberGeneratorStub{err: errors.New("number failed")})
	_, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: "inv_001"})
	if err == nil || !strings.Contains(err.Error(), "number failed") {
		t.Fatalf("IssueDraft() error = %v, want number failure", err)
	}
	if invoices.updateInvoice != nil {
		t.Fatalf("Update called unexpectedly: %#v", invoices.updateInvoice)
	}
	if invoice.Status != core.InvoiceStatusDraft {
		t.Fatalf("invoice status = %q, want draft", invoice.Status)
	}
}

// -- Discard (unified) --

func TestInvoiceServiceDiscard_DraftHardDelete(t *testing.T) {
	t.Parallel()

	entry1, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work A", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)})
	entry2, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work B", Hours: mustHours(30000), Billable: true, Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)})
	_ = entry1.AssignToInvoice("inv_001")
	_ = entry2.AssignToInvoice("inv_001")
	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry1.ID, UnitRate: rate})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry2.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1, line2}})
	invoices := &invoiceStoreStub{getByIDRes: &invoice}

	svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, agreementsForInvoice(), profilesForInvoice())

	result, err := svc.Discard(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if result.WasSoftDiscard {
		t.Fatal("WasSoftDiscard = true, want false for draft")
	}
	if invoices.deleteID != invoice.ID {
		t.Fatalf("Delete called with %q, want %q", invoices.deleteID, invoice.ID)
	}
	// Entry unlocking is handled atomically by the store layer (integration-tested).
}

func TestInvoiceServiceDiscard_IssuedSoftDiscard(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	entry, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: hours, Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)})
	_ = entry.AssignToInvoice("inv_001")
	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusIssued, Currency: "USD", Lines: []core.InvoiceLine{line}})
	invoice.InvoiceNumber = "INV-2026-0001"
	invoices := &invoiceStoreStub{getByIDRes: &invoice}

	svc := NewInvoiceService(invoices, &timeEntryStoreStub{getByIDRes: &entry}, agreementsForInvoice(), profilesForInvoice())

	result, err := svc.Discard(context.Background(), invoice.ID)
	if err != nil {
		t.Fatalf("Discard() error = %v", err)
	}
	if !result.WasSoftDiscard {
		t.Fatal("WasSoftDiscard = false, want true for issued")
	}
	if result.InvoiceNumber != "INV-2026-0001" {
		t.Fatalf("InvoiceNumber = %q, want INV-2026-0001", result.InvoiceNumber)
	}
	if invoices.deleteID != "" {
		t.Fatalf("Delete called unexpectedly with %q", invoices.deleteID)
	}
	if invoices.updateInvoice == nil {
		t.Fatal("Update was not called for soft-discard")
	}
	if invoices.updateInvoice.Status != core.InvoiceStatusDiscarded {
		t.Fatalf("updated status = %q, want discarded", invoices.updateInvoice.Status)
	}
	// Entry should remain locked.
	if err := entry.Update("should fail", hours); err == nil {
		t.Fatal("entry should remain locked after soft-discard")
	}
}

func TestInvoiceServiceDiscard_RejectsAlreadyDiscarded(t *testing.T) {
	t.Parallel()

	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: "te_001", UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusDiscarded, Currency: "USD", Lines: []core.InvoiceLine{line}})
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, agreementsForInvoice(), profilesForInvoice())

	_, err := svc.Discard(context.Background(), invoice.ID)
	if err == nil {
		t.Fatal("Discard() error = nil, want already-discarded rejection")
	}
	if !strings.Contains(err.Error(), "already discarded") {
		t.Fatalf("Discard() error = %v, want already discarded", err)
	}
}

type multiEntryStoreStub struct {
	timeEntryStoreStub
	second   *core.TimeEntry
	secondID string
}

func (s *multiEntryStoreStub) GetByID(ctx context.Context, id string) (*core.TimeEntry, error) {
	if id == s.secondID {
		return s.second, nil
	}
	return s.timeEntryStoreStub.GetByID(ctx, id)
}

func TestInvoiceServiceGetInvoice_HappyPath(t *testing.T) {
	t.Parallel()

	hours := mustHours(15000)
	entry1 := mustIssueDraftEntry("te_001", "Work A", hours)
	entry2 := mustIssueDraftEntry("te_002", "Work B", mustHours(30000))
	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_001", ServiceAgreementID: "sa_1", TimeEntryID: "te_001", UnitRate: rate})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_001", ServiceAgreementID: "sa_1", TimeEntryID: "te_002", UnitRate: rate})
	inv, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_1", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1, line2}})
	inv.ID = "inv_001"

	entries := &multiEntryStoreStub{
		timeEntryStoreStub: timeEntryStoreStub{getByIDRes: entry1},
		second:             entry2,
		secondID:           "te_002",
	}
	invoices := &invoiceStoreStub{getByIDRes: &inv}
	svc := NewInvoiceService(invoices, entries, nil, nil)

	dto, err := svc.GetInvoice(context.Background(), "inv_001")
	if err != nil {
		t.Fatalf("GetInvoice() error = %v", err)
	}
	if dto.ID != "inv_001" {
		t.Fatalf("ID = %q, want inv_001", dto.ID)
	}
	if len(dto.Lines) != 2 {
		t.Fatalf("len(Lines) = %d, want 2", len(dto.Lines))
	}
	if dto.Lines[0].Description != "Work A" {
		t.Fatalf("Lines[0].Description = %q, want Work A", dto.Lines[0].Description)
	}
	if dto.Lines[1].Description != "Work B" {
		t.Fatalf("Lines[1].Description = %q, want Work B", dto.Lines[1].Description)
	}
}

func TestInvoiceServiceGetInvoice_NotFound(t *testing.T) {
	t.Parallel()

	invoices := &invoiceStoreStub{getByIDErr: ErrInvoiceNotFound}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{}, nil, nil)

	_, err := svc.GetInvoice(context.Background(), "inv_999")
	if !errors.Is(err, ErrInvoiceNotFound) {
		t.Fatalf("GetInvoice() error = %v, want ErrInvoiceNotFound", err)
	}
}

func TestInvoiceServiceListInvoices_HappyPath(t *testing.T) {
	t.Parallel()

	summaries := []core.InvoiceSummary{
		{ID: "inv_001", InvoiceNumber: "INV-001", Status: core.InvoiceStatusIssued, Currency: "USD", GrandTotal: 5000, PeriodStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC), DueDate: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)},
		{ID: "inv_002", InvoiceNumber: "", Status: core.InvoiceStatusDraft, Currency: "USD", GrandTotal: 3000},
	}
	invoices := &invoiceStoreStub{listByCustomerRes: summaries}
	svc := NewInvoiceService(invoices, nil, nil, nil)

	dtos, err := svc.ListInvoices(context.Background(), "cus_1", "")
	if err != nil {
		t.Fatalf("ListInvoices() error = %v", err)
	}
	if len(dtos) != 2 {
		t.Fatalf("len(dtos) = %d, want 2", len(dtos))
	}
	if dtos[0].ID != "inv_001" {
		t.Fatalf("dtos[0].ID = %q, want inv_001", dtos[0].ID)
	}
	if dtos[1].Status != "draft" {
		t.Fatalf("dtos[1].Status = %q, want draft", dtos[1].Status)
	}
	if dtos[0].PeriodStart != "2026-04-01T00:00:00Z" || dtos[0].PeriodEnd != "2026-04-30T00:00:00Z" || dtos[0].DueDate != "2026-05-15T00:00:00Z" {
		t.Fatalf("summary metadata = (%q,%q,%q)", dtos[0].PeriodStart, dtos[0].PeriodEnd, dtos[0].DueDate)
	}
}

func TestInvoiceServiceListInvoices_StatusFilter(t *testing.T) {
	t.Parallel()

	summaries := []core.InvoiceSummary{
		{ID: "inv_002", Status: core.InvoiceStatusDraft, Currency: "USD", GrandTotal: 3000},
	}
	invoices := &invoiceStoreStub{listByCustomerRes: summaries, listByCustomerStatusFilter: "draft"}
	svc := NewInvoiceService(invoices, nil, nil, nil)

	dtos, err := svc.ListInvoices(context.Background(), "cus_1", "draft")
	if err != nil {
		t.Fatalf("ListInvoices() error = %v", err)
	}
	if len(dtos) != 1 {
		t.Fatalf("len(dtos) = %d, want 1", len(dtos))
	}
	if dtos[0].Status != "draft" {
		t.Fatalf("dtos[0].Status = %q, want draft", dtos[0].Status)
	}
}

func TestInvoiceServiceListInvoices_InvalidStatusFilter(t *testing.T) {
	t.Parallel()

	invoices := &invoiceStoreStub{}
	svc := NewInvoiceService(invoices, nil, nil, nil)

	_, err := svc.ListInvoices(context.Background(), "cus_1", "pending")
	if !errors.Is(err, ErrInvalidStatusFilter) {
		t.Fatalf("ListInvoices() error = %v, want ErrInvalidStatusFilter", err)
	}

	_, err = svc.ListInvoices(context.Background(), "cus_1", "DRAFT")
	if !errors.Is(err, ErrInvalidStatusFilter) {
		t.Fatalf("ListInvoices() with uppercase 'DRAFT' error = %v, want ErrInvalidStatusFilter", err)
	}
}

func TestInvoiceServiceListInvoices_Empty(t *testing.T) {
	t.Parallel()

	invoices := &invoiceStoreStub{listByCustomerRes: nil}
	svc := NewInvoiceService(invoices, nil, nil, nil)

	dtos, err := svc.ListInvoices(context.Background(), "cus_999", "")
	if err != nil {
		t.Fatalf("ListInvoices() error = %v", err)
	}
	if len(dtos) != 0 {
		t.Fatalf("len(dtos) = %d, want 0", len(dtos))
	}
}

func agreementsForInvoice() *serviceAgreementStoreForTimeEntry {
	return &serviceAgreementStoreForTimeEntry{getByIDRes: activeAgreement()}
}

func profilesForInvoice() *customerProfileStoreForTimeEntry {
	return &customerProfileStoreForTimeEntry{getByIDRes: activeProfile()}
}

func inactiveProfile() *core.CustomerProfile {
	profile := activeProfile()
	profile.Status = core.CustomerProfileStatusInactive
	return profile
}

func mustIssueDraftEntry(id, description string, hours core.Hours) *core.TimeEntry {
	return &core.TimeEntry{ID: id, Description: description, Hours: hours}
}

func newIssueDraftService(t *testing.T, invoice *core.Invoice, entry *core.TimeEntry, numbers *invoiceNumberGeneratorStub) (InvoiceService, *invoiceStoreStub, *timeEntryStoreStub) {
	t.Helper()
	if invoice == nil {
		invoice = &core.Invoice{}
	}
	if entry == nil {
		entry = &core.TimeEntry{}
	}
	invoices := &invoiceStoreStub{getByIDRes: invoice}
	entries := &timeEntryStoreStub{getByIDRes: entry}
	return NewInvoiceService(invoices, entries, nil, nil, numbers), invoices, entries
}
