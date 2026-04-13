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
	createDraftInvoice *core.Invoice
	createDraftEntries []*core.TimeEntry
	updateInvoice      *core.Invoice
	updateErr          error
	createDraftErr     error
	getByIDRes         *core.Invoice
	getByIDErr         error
	deleteID           string
	deleteErr          error
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

type invoiceNumberGeneratorStub struct {
	next      string
	err       error
	callCount int
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

	hours1, _ := core.NewHours(15000)
	hours2, _ := core.NewHours(30000)
	entry1, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work A", Hours: hours1, Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)})
	entry2, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work B", Hours: hours2, Billable: true, Date: time.Date(2026, 4, 9, 0, 0, 0, 0, time.UTC)})
	_ = entry1.AssignToInvoice("inv_001")
	_ = entry2.AssignToInvoice("inv_001")
	rate, _ := core.NewMoney(10000, "USD")
	line1, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry1.ID, UnitRate: rate})
	line2, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry2.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusDraft, Currency: "USD", Lines: []core.InvoiceLine{line1, line2}})
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	entries := &timeEntryStoreStub{getByIDRes: &entry1}
	svc := NewInvoiceService(invoices, entries, agreementsForInvoice(), profilesForInvoice())

	// Override stores with deterministic behavior for both entry fetches.
	svc.entries = &multiEntryStoreStub{timeEntryStoreStub: timeEntryStoreStub{getByIDRes: &entry1}, second: &entry2, secondID: entry2.ID}

	if err := svc.DiscardDraft(context.Background(), invoice.ID); err != nil {
		t.Fatalf("DiscardDraft() error = %v", err)
	}
	if invoices.deleteID != invoice.ID {
		t.Fatalf("Delete called with %q, want %q", invoices.deleteID, invoice.ID)
	}
	if err := entry1.Update("updated", hours1); err != nil {
		t.Fatalf("entry1 should be unlocked after discard: %v", err)
	}
	if err := entry2.Update("updated", hours2); err != nil {
		t.Fatalf("entry2 should be unlocked after discard: %v", err)
	}
}

func TestInvoiceServiceDiscardDraft_RejectsIssuedInvoice(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	entry, _ := core.NewTimeEntry(core.TimeEntryParams{CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: hours, Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)})
	rate, _ := core.NewMoney(10000, "USD")
	line, _ := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: "inv_seed", ServiceAgreementID: "sa_xyz789", TimeEntryID: entry.ID, UnitRate: rate})
	invoice, _ := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: core.InvoiceStatusIssued, Currency: "USD", Lines: []core.InvoiceLine{line}})
	invoices := &invoiceStoreStub{getByIDRes: &invoice}
	svc := NewInvoiceService(invoices, &timeEntryStoreStub{getByIDRes: &entry}, agreementsForInvoice(), profilesForInvoice())

	if err := svc.DiscardDraft(context.Background(), invoice.ID); err == nil {
		t.Fatal("DiscardDraft() error = nil, want non-draft rejection")
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
