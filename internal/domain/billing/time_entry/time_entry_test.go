package timeentry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
	"github.com/Carlos0934/billar/internal/domain/billing/time_entry"
)

func TestNewTimeEntryCreatesBillableUninvoicedEntry(t *testing.T) {
	entryID := mustTimeEntryID(t, "time-123")
	customerID := mustCustomerID(t, "cust-123")
	agreementID := mustServiceAgreementID(t, "agr-123")
	workDate := mustWorkDate(t, time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC))
	hours := mustHours(t, 12500)

	entry, err := timeentry.New(timeentry.NewTimeEntryParams{
		ID:                 entryID,
		CustomerID:         customerID,
		ServiceAgreementID: agreementID,
		WorkDate:           workDate,
		Hours:              hours,
		Description:        "Investigated invoice discrepancy",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got := entry.ID().String(); got != entryID.String() {
		t.Fatalf("ID() = %q, want %q", got, entryID.String())
	}

	if got := entry.CustomerID().String(); got != customerID.String() {
		t.Fatalf("CustomerID() = %q, want %q", got, customerID.String())
	}

	if got := entry.ServiceAgreementID().String(); got != agreementID.String() {
		t.Fatalf("ServiceAgreementID() = %q, want %q", got, agreementID.String())
	}

	if got := entry.WorkDate().Time(); !got.Equal(workDate.Time()) {
		t.Fatalf("WorkDate() = %v, want %v", got, workDate.Time())
	}

	if got := entry.Hours().Value(); got != hours.Value() {
		t.Fatalf("Hours() = %d, want %d", got, hours.Value())
	}

	if got := entry.Description(); got != "Investigated invoice discrepancy" {
		t.Fatalf("Description() = %q, want %q", got, "Investigated invoice discrepancy")
	}

	if !entry.Billable() {
		t.Fatal("expected new entry to be billable")
	}

	if entry.Invoiced() {
		t.Fatal("expected new entry to be uninvoiced")
	}

	if !entry.InvoiceID().IsZero() {
		t.Fatal("expected new entry to have empty invoice id")
	}
}

func TestNewTimeEntryRejectsNonPositiveHours(t *testing.T) {
	_, err := timeentry.New(timeentry.NewTimeEntryParams{
		ID:                 mustTimeEntryID(t, "time-123"),
		CustomerID:         mustCustomerID(t, "cust-123"),
		ServiceAgreementID: mustServiceAgreementID(t, "agr-123"),
		WorkDate:           mustWorkDate(t, time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)),
		Hours:              billingvalues.Hours{},
		Description:        "Investigated invoice discrepancy",
	})
	if err == nil {
		t.Fatal("expected error for non-positive hours")
	}
	if !errors.Is(err, billingvalues.ErrHoursMustBePositive) {
		t.Fatalf("err = %v, want %v", err, billingvalues.ErrHoursMustBePositive)
	}
}

func TestTimeEntryUpdateHoursRejectsNonPositiveHours(t *testing.T) {
	entry := newTimeEntry(t)

	err := entry.UpdateHours(billingvalues.Hours{})
	if err == nil {
		t.Fatal("expected error for non-positive hours")
	}
	if !errors.Is(err, billingvalues.ErrHoursMustBePositive) {
		t.Fatalf("err = %v, want %v", err, billingvalues.ErrHoursMustBePositive)
	}

	if got := entry.Hours().Value(); got != 12500 {
		t.Fatalf("Hours() = %d, want %d", got, 12500)
	}
}

func TestTimeEntryAllowsMutationsBeforeInvoiceAssignment(t *testing.T) {
	entry := newTimeEntry(t)

	updatedHours := mustHours(t, 25000)
	if err := entry.UpdateHours(updatedHours); err != nil {
		t.Fatalf("UpdateHours() error = %v", err)
	}

	if got := entry.Hours().Value(); got != updatedHours.Value() {
		t.Fatalf("Hours() = %d, want %d", got, updatedHours.Value())
	}

	if err := entry.UpdateDescription("  Pairing with client on payment terms  "); err != nil {
		t.Fatalf("UpdateDescription() error = %v", err)
	}

	if got := entry.Description(); got != "Pairing with client on payment terms" {
		t.Fatalf("Description() = %q, want %q", got, "Pairing with client on payment terms")
	}

	if err := entry.SetBillable(false); err != nil {
		t.Fatalf("SetBillable(false) error = %v", err)
	}

	if entry.Billable() {
		t.Fatal("expected entry to be non-billable after SetBillable(false)")
	}

	if err := entry.SetBillable(true); err != nil {
		t.Fatalf("SetBillable(true) error = %v", err)
	}

	if !entry.Billable() {
		t.Fatal("expected entry to be billable after SetBillable(true)")
	}
}

func TestTimeEntryAssignsInvoiceOnceForBillableEntry(t *testing.T) {
	entry := newTimeEntry(t)
	invoiceID := mustInvoiceID(t, "inv-123")

	if err := entry.AssignInvoice(invoiceID); err != nil {
		t.Fatalf("AssignInvoice() error = %v", err)
	}

	if !entry.Invoiced() {
		t.Fatal("expected entry to be invoiced after assignment")
	}

	if got := entry.InvoiceID().String(); got != invoiceID.String() {
		t.Fatalf("InvoiceID() = %q, want %q", got, invoiceID.String())
	}

	if !entry.Locked() {
		t.Fatal("expected entry to be locked after invoice assignment")
	}
}

func TestTimeEntryRejectsInvoiceAssignmentWhenNonBillable(t *testing.T) {
	entry := newTimeEntry(t)
	if err := entry.SetBillable(false); err != nil {
		t.Fatalf("SetBillable(false) error = %v", err)
	}

	err := entry.AssignInvoice(mustInvoiceID(t, "inv-123"))
	if err == nil {
		t.Fatal("expected error when assigning invoice to non-billable entry")
	}
	if !errors.Is(err, timeentry.ErrInvoiceAssignmentRequiresBillable) {
		t.Fatalf("err = %v, want %v", err, timeentry.ErrInvoiceAssignmentRequiresBillable)
	}

	if entry.Invoiced() {
		t.Fatal("expected entry to remain uninvoiced")
	}
}

func TestTimeEntryRejectsDuplicateInvoiceAssignment(t *testing.T) {
	entry := newTimeEntry(t)
	if err := entry.AssignInvoice(mustInvoiceID(t, "inv-123")); err != nil {
		t.Fatalf("AssignInvoice() error = %v", err)
	}

	err := entry.AssignInvoice(mustInvoiceID(t, "inv-456"))
	if err == nil {
		t.Fatal("expected error when assigning invoice twice")
	}
	if !errors.Is(err, timeentry.ErrTimeEntryAlreadyInvoiced) {
		t.Fatalf("err = %v, want %v", err, timeentry.ErrTimeEntryAlreadyInvoiced)
	}

	if got := entry.InvoiceID().String(); got != "inv-123" {
		t.Fatalf("InvoiceID() = %q, want %q", got, "inv-123")
	}
}

func TestTimeEntryLocksFinancialFactsAfterInvoicing(t *testing.T) {
	entry := newTimeEntry(t)
	if err := entry.AssignInvoice(mustInvoiceID(t, "inv-123")); err != nil {
		t.Fatalf("AssignInvoice() error = %v", err)
	}

	if err := entry.UpdateHours(mustHours(t, 25000)); !errors.Is(err, timeentry.ErrTimeEntryLocked) {
		t.Fatalf("UpdateHours() err = %v, want %v", err, timeentry.ErrTimeEntryLocked)
	}

	if err := entry.UpdateDescription("changed after invoice"); !errors.Is(err, timeentry.ErrTimeEntryLocked) {
		t.Fatalf("UpdateDescription() err = %v, want %v", err, timeentry.ErrTimeEntryLocked)
	}

	if err := entry.SetBillable(false); !errors.Is(err, timeentry.ErrTimeEntryLocked) {
		t.Fatalf("SetBillable() err = %v, want %v", err, timeentry.ErrTimeEntryLocked)
	}

	if got := entry.Hours().Value(); got != 12500 {
		t.Fatalf("Hours() = %d, want %d", got, 12500)
	}

	if got := entry.Description(); got != "Investigated invoice discrepancy" {
		t.Fatalf("Description() = %q, want %q", got, "Investigated invoice discrepancy")
	}

	if !entry.Billable() {
		t.Fatal("expected billable flag to remain unchanged after lock")
	}
}

func TestTimeEntryPreservesInvoiceIDAfterClearOrReplaceAttempts(t *testing.T) {
	entry := newTimeEntry(t)
	if err := entry.AssignInvoice(mustInvoiceID(t, "inv-123")); err != nil {
		t.Fatalf("AssignInvoice() error = %v", err)
	}

	if err := entry.AssignInvoice(billingvalues.InvoiceID{}); !errors.Is(err, timeentry.ErrTimeEntryAlreadyInvoiced) {
		t.Fatalf("AssignInvoice(zero) err = %v, want %v", err, timeentry.ErrTimeEntryAlreadyInvoiced)
	}

	if err := entry.AssignInvoice(mustInvoiceID(t, "inv-456")); !errors.Is(err, timeentry.ErrTimeEntryAlreadyInvoiced) {
		t.Fatalf("AssignInvoice(replace) err = %v, want %v", err, timeentry.ErrTimeEntryAlreadyInvoiced)
	}

	if got := entry.InvoiceID().String(); got != "inv-123" {
		t.Fatalf("InvoiceID() = %q, want %q", got, "inv-123")
	}
}

func newTimeEntry(t *testing.T) *timeentry.TimeEntry {
	t.Helper()

	entry, err := timeentry.New(timeentry.NewTimeEntryParams{
		ID:                 mustTimeEntryID(t, "time-123"),
		CustomerID:         mustCustomerID(t, "cust-123"),
		ServiceAgreementID: mustServiceAgreementID(t, "agr-123"),
		WorkDate:           mustWorkDate(t, time.Date(2026, time.March, 30, 12, 0, 0, 0, time.UTC)),
		Hours:              mustHours(t, 12500),
		Description:        "Investigated invoice discrepancy",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return entry
}

func mustInvoiceID(t *testing.T, value string) billingvalues.InvoiceID {
	t.Helper()

	id, err := billingvalues.NewInvoiceID(value)
	if err != nil {
		t.Fatalf("NewInvoiceID() error = %v", err)
	}

	return id
}

func mustTimeEntryID(t *testing.T, value string) billingvalues.TimeEntryID {
	t.Helper()

	id, err := billingvalues.NewTimeEntryID(value)
	if err != nil {
		t.Fatalf("NewTimeEntryID() error = %v", err)
	}

	return id
}

func mustCustomerID(t *testing.T, value string) billingvalues.CustomerID {
	t.Helper()

	id, err := billingvalues.NewCustomerID(value)
	if err != nil {
		t.Fatalf("NewCustomerID() error = %v", err)
	}

	return id
}

func mustServiceAgreementID(t *testing.T, value string) billingvalues.ServiceAgreementID {
	t.Helper()

	id, err := billingvalues.NewServiceAgreementID(value)
	if err != nil {
		t.Fatalf("NewServiceAgreementID() error = %v", err)
	}

	return id
}

func mustHours(t *testing.T, value int64) billingvalues.Hours {
	t.Helper()

	hours, err := billingvalues.NewHours(value)
	if err != nil {
		t.Fatalf("NewHours() error = %v", err)
	}

	return hours
}

func mustWorkDate(t *testing.T, value time.Time) timeentry.WorkDate {
	t.Helper()

	workDate, err := timeentry.NewWorkDate(value)
	if err != nil {
		t.Fatalf("NewWorkDate() error = %v", err)
	}

	return workDate
}
