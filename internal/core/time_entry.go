package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	timeEntryIDPrefix   = "te_"
	timeEntryIDBytes    = 16
	timeEntryIDHexChars = 32
)

// Sentinel errors for TimeEntry domain invariants.
var (
	ErrInvalidTimeEntry        = errors.New("time entry is invalid")
	ErrMissingServiceAgreement = errors.New("service agreement is required for billable time entries")
	ErrTimeEntryLocked         = errors.New("time entry is locked: it has been assigned to an invoice")
)

// TimeEntry represents a recorded unit of work for a customer.
type TimeEntry struct {
	ID                 string
	CustomerProfileID  string
	ServiceAgreementID string
	Description        string
	Hours              Hours
	Billable           bool
	InvoiceID          string
	Date               time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TimeEntryParams holds the inputs for creating a new TimeEntry.
type TimeEntryParams struct {
	CustomerProfileID  string
	ServiceAgreementID string
	Description        string
	Hours              Hours
	Billable           bool
	Date               time.Time
}

// NewTimeEntry constructs a validated TimeEntry from params.
// All domain invariants are enforced at construction time.
func NewTimeEntry(params TimeEntryParams) (TimeEntry, error) {
	if strings.TrimSpace(params.CustomerProfileID) == "" {
		return TimeEntry{}, errors.New("time entry customer profile id is required")
	}
	if strings.TrimSpace(params.Description) == "" {
		return TimeEntry{}, errors.New("time entry description is required")
	}
	if !params.Hours.IsPositive() {
		return TimeEntry{}, errors.New("time entry hours must be positive")
	}
	if params.Billable && strings.TrimSpace(params.ServiceAgreementID) == "" {
		return TimeEntry{}, ErrMissingServiceAgreement
	}

	now := time.Now().UTC()
	entry := TimeEntry{
		ID:                 generateTimeEntryID(),
		CustomerProfileID:  strings.TrimSpace(params.CustomerProfileID),
		ServiceAgreementID: strings.TrimSpace(params.ServiceAgreementID),
		Description:        strings.TrimSpace(params.Description),
		Hours:              params.Hours,
		Billable:           params.Billable,
		InvoiceID:          "",
		Date:               params.Date,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if entry.ID == "" {
		return TimeEntry{}, errors.New("failed to generate time entry id")
	}

	return entry, nil
}

// locked reports whether this entry is immutable due to invoice assignment.
func (t *TimeEntry) locked() bool {
	return t.InvoiceID != ""
}

// Update modifies the description and hours of the entry.
// Returns ErrTimeEntryLocked if the entry has been assigned to an invoice.
// Returns an error if description is blank or hours are non-positive.
func (t *TimeEntry) Update(description string, hours Hours) error {
	if t.locked() {
		return ErrTimeEntryLocked
	}
	if strings.TrimSpace(description) == "" {
		return errors.New("time entry description is required")
	}
	if !hours.IsPositive() {
		return errors.New("time entry hours must be positive")
	}
	t.Description = strings.TrimSpace(description)
	t.Hours = hours
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// AssignToInvoice sets the invoice ID on this entry, locking it from further mutations.
// The invoice ID must be non-blank.
func (t *TimeEntry) AssignToInvoice(invoiceID string) error {
	if strings.TrimSpace(invoiceID) == "" {
		return errors.New("invoice id is required")
	}
	t.InvoiceID = strings.TrimSpace(invoiceID)
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// UnassignFromInvoice clears the invoice ID, making the entry mutable again.
func (t *TimeEntry) UnassignFromInvoice() {
	t.InvoiceID = ""
	t.UpdatedAt = time.Now().UTC()
}

func generateTimeEntryID() string {
	buf := make([]byte, timeEntryIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	encoded := hex.EncodeToString(buf)
	if len(encoded) != timeEntryIDHexChars {
		return ""
	}
	return timeEntryIDPrefix + encoded
}
