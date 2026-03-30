package timeentry

import (
	"strings"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

type NewTimeEntryParams struct {
	ID                 billingvalues.TimeEntryID
	CustomerID         billingvalues.CustomerID
	ServiceAgreementID billingvalues.ServiceAgreementID
	WorkDate           WorkDate
	Hours              billingvalues.Hours
	Description        string
}

type TimeEntry struct {
	id                 billingvalues.TimeEntryID
	customerID         billingvalues.CustomerID
	serviceAgreementID billingvalues.ServiceAgreementID
	workDate           WorkDate
	hours              billingvalues.Hours
	description        string
	billable           bool
	invoiced           bool
	invoiceID          billingvalues.InvoiceID
}

func New(params NewTimeEntryParams) (*TimeEntry, error) {
	if params.ID.IsZero() {
		return nil, ErrTimeEntryIDRequired
	}
	if params.CustomerID.IsZero() {
		return nil, ErrCustomerIDRequired
	}
	if params.ServiceAgreementID.IsZero() {
		return nil, ErrServiceAgreementIDRequired
	}
	if params.WorkDate.IsZero() {
		return nil, ErrWorkDateRequired
	}
	if err := validateHours(params.Hours); err != nil {
		return nil, err
	}

	entry := &TimeEntry{
		id:                 params.ID,
		customerID:         params.CustomerID,
		serviceAgreementID: params.ServiceAgreementID,
		workDate:           params.WorkDate,
		hours:              params.Hours,
		description:        normalizeDescription(params.Description),
		billable:           true,
		invoiced:           false,
	}

	if err := entry.validateState(); err != nil {
		return nil, err
	}

	return entry, nil
}

func (entry *TimeEntry) ID() billingvalues.TimeEntryID {
	return entry.id
}

func (entry *TimeEntry) CustomerID() billingvalues.CustomerID {
	return entry.customerID
}

func (entry *TimeEntry) ServiceAgreementID() billingvalues.ServiceAgreementID {
	return entry.serviceAgreementID
}

func (entry *TimeEntry) WorkDate() WorkDate {
	return entry.workDate
}

func (entry *TimeEntry) Hours() billingvalues.Hours {
	return entry.hours
}

func (entry *TimeEntry) Description() string {
	return entry.description
}

func (entry *TimeEntry) Billable() bool {
	return entry.billable
}

func (entry *TimeEntry) Invoiced() bool {
	return entry.invoiced
}

func (entry *TimeEntry) InvoiceID() billingvalues.InvoiceID {
	return entry.invoiceID
}

func (entry *TimeEntry) Locked() bool {
	return entry.invoiced
}

func (entry *TimeEntry) UpdateHours(hours billingvalues.Hours) error {
	if err := entry.ensureMutable(); err != nil {
		return err
	}
	if err := validateHours(hours); err != nil {
		return err
	}

	entry.hours = hours
	return nil
}

func (entry *TimeEntry) UpdateDescription(description string) error {
	if err := entry.ensureMutable(); err != nil {
		return err
	}

	entry.description = normalizeDescription(description)
	return nil
}

func (entry *TimeEntry) SetBillable(billable bool) error {
	if err := entry.ensureMutable(); err != nil {
		return err
	}

	entry.billable = billable
	return nil
}

func (entry *TimeEntry) AssignInvoice(id billingvalues.InvoiceID) error {
	if err := entry.ensureInvoiceAssignable(id); err != nil {
		return err
	}

	entry.invoiced = true
	entry.invoiceID = id

	return entry.validateState()
}

func (entry *TimeEntry) ensureInvoiceAssignable(id billingvalues.InvoiceID) error {
	if err := entry.validateState(); err != nil {
		return err
	}
	if entry.invoiced {
		return ErrTimeEntryAlreadyInvoiced
	}
	if !entry.billable {
		return ErrInvoiceAssignmentRequiresBillable
	}
	if id.IsZero() {
		return billingvalues.ErrInvoiceIDRequired
	}

	return nil
}

func (entry *TimeEntry) ensureMutable() error {
	if err := entry.validateState(); err != nil {
		return err
	}
	if entry.Locked() {
		return ErrTimeEntryLocked
	}

	return nil
}

func (entry *TimeEntry) validateState() error {
	if entry.invoiced && !entry.hasInvoiceID() {
		return ErrInvoicedEntryRequiresInvoiceID
	}
	if !entry.invoiced && entry.hasInvoiceID() {
		return ErrInvoiceIDRequiresInvoicedEntry
	}

	return nil
}

func (entry *TimeEntry) hasInvoiceID() bool {
	return !entry.invoiceID.IsZero()
}

func normalizeDescription(description string) string {
	return strings.TrimSpace(description)
}

func validateHours(hours billingvalues.Hours) error {
	if !hours.IsPositive() {
		return billingvalues.ErrHoursMustBePositive
	}

	return nil
}
