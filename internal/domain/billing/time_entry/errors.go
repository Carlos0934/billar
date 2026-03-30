package timeentry

import "errors"

var (
	ErrTimeEntryIDRequired               = errors.New("timeentry: id is required")
	ErrCustomerIDRequired                = errors.New("timeentry: customer id is required")
	ErrServiceAgreementIDRequired        = errors.New("timeentry: service agreement id is required")
	ErrWorkDateRequired                  = errors.New("timeentry: work date is required")
	ErrTimeEntryLocked                   = errors.New("timeentry: invoiced financial facts are locked")
	ErrInvoiceAssignmentRequiresBillable = errors.New("timeentry: invoice assignment requires billable entry")
	ErrTimeEntryAlreadyInvoiced          = errors.New("timeentry: entry is already invoiced")
	ErrInvoicedEntryRequiresInvoiceID    = errors.New("timeentry: invoiced entry requires invoice id")
	ErrInvoiceIDRequiresInvoicedEntry    = errors.New("timeentry: invoice id requires invoiced entry")
)
