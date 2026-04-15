package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

func TestInvoiceServiceCreateDraftFromUnbilled_NegativePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(*customerProfileStoreForTimeEntry, *serviceAgreementStoreForTimeEntry, *timeEntryStoreStub, *invoiceStoreStub)
		want        string
		missingDeps bool
	}{
		{
			name: "blank customer profile id",
			setup: func(_ *customerProfileStoreForTimeEntry, _ *serviceAgreementStoreForTimeEntry, _ *timeEntryStoreStub, _ *invoiceStoreStub) {
			},
			want: "customer profile id is required",
		},
		{
			name:        "missing dependencies",
			want:        "invoice service dependencies are required",
			missingDeps: true,
		},
		{
			name: "customer profile not found",
			setup: func(profiles *customerProfileStoreForTimeEntry, _ *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				profiles.getByIDErr = ErrCustomerProfileNotFound
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
			},
			want: "customer profile not found",
		},
		{
			name: "inactive customer profile",
			setup: func(profiles *customerProfileStoreForTimeEntry, _ *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				profiles.getByIDRes = inactiveProfile()
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
			},
			want: "customer profile is inactive",
		},
		{
			name: "unbilled list failure",
			setup: func(_ *customerProfileStoreForTimeEntry, _ *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				entries.listUnbilledErr = errors.New("list failed")
			},
			want: "list failed",
		},
		{
			name: "time entry customer mismatch",
			setup: func(profiles *customerProfileStoreForTimeEntry, agreements *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				profiles.getByIDRes = activeProfile()
				agreements.getByIDRes = activeAgreement()
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_other", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
			},
			want: "time entry customer mismatch",
		},
		{
			name: "service agreement not found",
			setup: func(_ *customerProfileStoreForTimeEntry, agreements *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				agreements.getByIDErr = ErrServiceAgreementNotFound
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_missing", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
			},
			want: "service agreement not found",
		},
		{
			name: "inactive service agreement",
			setup: func(_ *customerProfileStoreForTimeEntry, agreements *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				agreements.getByIDRes = inactiveAgreement()
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
			},
			want: "service agreement is inactive",
		},
		{
			name: "service agreement currency mismatch",
			setup: func(profiles *customerProfileStoreForTimeEntry, agreements *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, _ *invoiceStoreStub) {
				profiles.getByIDRes = activeProfile()
				agreements.getByIDRes = &core.ServiceAgreement{ID: "sa_xyz789", CustomerProfileID: "cus_abc123", Name: "Support", BillingMode: core.BillingModeHourly, HourlyRate: 1000, Currency: "EUR", Active: true}
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
			},
			want: "must match customer currency",
		},
		{
			name: "invoice save failure",
			setup: func(_ *customerProfileStoreForTimeEntry, _ *serviceAgreementStoreForTimeEntry, entries *timeEntryStoreStub, invoices *invoiceStoreStub) {
				entries.listUnbilledRes = []core.TimeEntry{{ID: "te_001", CustomerProfileID: "cus_abc123", ServiceAgreementID: "sa_xyz789", Description: "Work", Hours: mustHours(15000), Billable: true, Date: time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)}}
				invoices.createDraftErr = errors.New("save failed")
			},
			want: "save failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceService
			if tt.missingDeps {
				svc = NewInvoiceService(nil, nil, nil, nil)
			} else {
				profiles, agreements, entries, invoices := makeInvoiceServiceFixtures()
				tt.setup(profiles, agreements, entries, invoices)
				svc = NewInvoiceService(invoices, entries, agreements, profiles)
			}

			cmd := CreateDraftFromUnbilledCommand{CustomerProfileID: "cus_abc123"}
			if tt.name == "blank customer profile id" {
				cmd.CustomerProfileID = ""
			}

			_, err := svc.CreateDraftFromUnbilled(context.Background(), cmd)
			if err == nil || !errorMatches(err, tt.want) {
				t.Fatalf("CreateDraftFromUnbilled() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInvoiceServiceDiscardDraft_FailurePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setup       func(*invoiceStoreStub)
		want        string
		missingDeps bool
	}{
		{
			name:        "missing dependencies",
			want:        "invoice service dependencies are required",
			missingDeps: true,
		},
		{
			name: "invoice lookup failure",
			setup: func(invoices *invoiceStoreStub) {
				invoices.getByIDErr = errors.New("get invoice failed")
			},
			want: "get invoice failed",
		},
		{
			name: "already discarded invoice",
			setup: func(invoices *invoiceStoreStub) {
				invoices.getByIDRes = invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusDiscarded)
			},
			want: "already discarded",
		},
		{
			name: "delete failure",
			setup: func(invoices *invoiceStoreStub) {
				invoices.getByIDRes = invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusDraft)
				invoices.deleteErr = errors.New("delete failed")
			},
			want: "delete failed",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var svc InvoiceService
			if tt.missingDeps {
				svc = NewInvoiceService(nil, nil, nil, nil)
			} else {
				invoices := &invoiceStoreStub{}
				tt.setup(invoices)
				svc = NewInvoiceService(invoices, &timeEntryStoreStub{}, agreementsForInvoice(), profilesForInvoice())
			}

			err := svc.DiscardDraft(context.Background(), "inv_001")
			if err == nil || !errorMatches(err, tt.want) {
				t.Fatalf("DiscardDraft() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestInvoiceServiceIssueDraft_FailurePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(*invoiceStoreStub)
		want  string
	}{
		{name: "missing invoice id", want: "invoice id is required"},
		{name: "invoice lookup failure", setup: func(invoices *invoiceStoreStub) { invoices.getByIDErr = errors.New("get failed") }, want: "get failed"},
		{name: "non draft invoice", setup: func(invoices *invoiceStoreStub) {
			invoices.getByIDRes = invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusIssued)
		}, want: "invoice is not draft"},
		{name: "number generator failure", setup: func(invoices *invoiceStoreStub) {
			invoices.getByIDRes = invoiceWithSingleLine("inv_001", "te_001", core.InvoiceStatusDraft)
		}, want: "number failed"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			invoices := &invoiceStoreStub{}
			entries := &timeEntryStoreStub{getByIDRes: mustIssueDraftEntry("te_001", "Work", mustHours(15000))}
			if tt.setup != nil {
				tt.setup(invoices)
			}
			numbers := &invoiceNumberGeneratorStub{next: "INV-2026-0001"}
			if tt.name == "number generator failure" {
				numbers = &invoiceNumberGeneratorStub{err: errors.New("number failed")}
			}
			svc := NewInvoiceService(invoices, entries, nil, nil, numbers)
			invoiceID := "inv_001"
			if tt.name == "missing invoice id" {
				invoiceID = ""
			}
			_, err := svc.IssueDraft(context.Background(), IssueInvoiceCommand{InvoiceID: invoiceID})
			if err == nil || !errorMatches(err, tt.want) {
				t.Fatalf("IssueDraft() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func mustHours(amount int64) core.Hours {
	h, err := core.NewHours(amount)
	if err != nil {
		panic(err)
	}
	return h
}

func mustTimeEntry(id, customerID, agreementID string, hours core.Hours) *core.TimeEntry {
	entry := mustIssueDraftEntry(id, "Work", hours)
	entry.CustomerProfileID = customerID
	entry.ServiceAgreementID = agreementID
	entry.Billable = true
	entry.Date = time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	return entry
}

func invoiceWithSingleLine(id, timeEntryID string, status core.InvoiceStatus) *core.Invoice {
	line, err := core.NewInvoiceLine(core.InvoiceLineParams{InvoiceID: id, ServiceAgreementID: "sa_xyz789", TimeEntryID: timeEntryID, UnitRate: mustMoney(1000, "USD")})
	if err != nil {
		panic(err)
	}
	invoice, err := core.NewInvoice(core.InvoiceParams{CustomerID: "cus_abc123", Status: status, Currency: "USD", Lines: []core.InvoiceLine{line}})
	if err != nil {
		panic(err)
	}
	invoice.ID = id
	return &invoice
}

func mustMoney(amount int64, currency string) core.Money {
	m, err := core.NewMoney(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

func errorMatches(err error, want string) bool {
	switch want {
	case "invoice service dependencies are required":
		return err.Error() == want
	case "customer profile not found":
		return errors.Is(err, ErrCustomerProfileNotFound)
	case "service agreement not found":
		return errors.Is(err, ErrServiceAgreementNotFound)
	case "service agreement is inactive":
		return errors.Is(err, ErrInactiveServiceAgreement)
	default:
		return strings.Contains(err.Error(), want)
	}
}
