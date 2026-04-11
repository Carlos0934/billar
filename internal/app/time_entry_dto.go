package app

import (
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

// TimeEntryDTO is the canonical read model returned from TimeEntryService operations.
type TimeEntryDTO struct {
	ID                 string `json:"id" toon:"id"`
	CustomerProfileID  string `json:"customer_profile_id" toon:"customer_profile_id"`
	ServiceAgreementID string `json:"service_agreement_id,omitempty" toon:"service_agreement_id"`
	Description        string `json:"description" toon:"description"`
	Hours              int64  `json:"hours" toon:"hours"`
	Billable           bool   `json:"billable" toon:"billable"`
	InvoiceID          string `json:"invoice_id,omitempty" toon:"invoice_id"`
	Date               string `json:"date" toon:"date"`
	CreatedAt          string `json:"created_at" toon:"created_at"`
	UpdatedAt          string `json:"updated_at" toon:"updated_at"`
}

// RecordTimeEntryCommand carries all inputs needed to record a new TimeEntry.
type RecordTimeEntryCommand struct {
	CustomerProfileID  string    `json:"customer_profile_id"`
	ServiceAgreementID string    `json:"service_agreement_id,omitempty"`
	Description        string    `json:"description"`
	Hours              int64     `json:"hours"`
	Billable           bool      `json:"billable"`
	Date               time.Time `json:"date"`
}

// UpdateTimeEntryCommand carries the updated fields for an existing TimeEntry.
type UpdateTimeEntryCommand struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Hours       int64  `json:"hours"`
}

func timeEntryToDTO(entry core.TimeEntry) TimeEntryDTO {
	return TimeEntryDTO{
		ID:                 entry.ID,
		CustomerProfileID:  entry.CustomerProfileID,
		ServiceAgreementID: entry.ServiceAgreementID,
		Description:        entry.Description,
		Hours:              int64(entry.Hours),
		Billable:           entry.Billable,
		InvoiceID:          entry.InvoiceID,
		Date:               formatTimeEntryDate(entry.Date),
		CreatedAt:          formatTimeEntryDate(entry.CreatedAt),
		UpdatedAt:          formatTimeEntryDate(entry.UpdatedAt),
	}
}

func formatTimeEntryDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
