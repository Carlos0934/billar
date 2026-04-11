package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/core"
)

// Sentinel errors for TimeEntryService operations.
var (
	ErrTimeEntryNotFound        = errors.New("time entry not found")
	ErrInactiveServiceAgreement = errors.New("service agreement is inactive")
)

// TimeEntryStore is the persistence port for TimeEntry entities.
type TimeEntryStore interface {
	Save(ctx context.Context, entry *core.TimeEntry) error
	GetByID(ctx context.Context, id string) (*core.TimeEntry, error)
	Delete(ctx context.Context, id string) error
	ListByCustomerProfile(ctx context.Context, customerID string) ([]core.TimeEntry, error)
	ListUnbilled(ctx context.Context, customerID string) ([]core.TimeEntry, error)
}

// TimeEntryService orchestrates recording, updating, and querying of TimeEntry entities.
type TimeEntryService struct {
	entries    TimeEntryStore
	profiles   CustomerProfileStore
	agreements ServiceAgreementStore
}

// NewTimeEntryService constructs a TimeEntryService with injected stores.
func NewTimeEntryService(entries TimeEntryStore, profiles CustomerProfileStore, agreements ServiceAgreementStore) TimeEntryService {
	return TimeEntryService{entries: entries, profiles: profiles, agreements: agreements}
}

// Record validates cross-entity dependencies, constructs a new TimeEntry, and persists it.
func (s TimeEntryService) Record(ctx context.Context, cmd RecordTimeEntryCommand) (TimeEntryDTO, error) {
	if strings.TrimSpace(cmd.ServiceAgreementID) == "" {
		return TimeEntryDTO{}, fmt.Errorf("record time entry: %w", core.ErrMissingServiceAgreement)
	}

	// Validate customer profile exists
	if _, err := s.profiles.GetByID(ctx, cmd.CustomerProfileID); err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return TimeEntryDTO{}, fmt.Errorf("record time entry: %w", ErrCustomerProfileNotFound)
		}
		return TimeEntryDTO{}, fmt.Errorf("record time entry: get customer profile: %w", err)
	}

	// Validate service agreement is active when billable
	if cmd.Billable {
		sa, err := s.agreements.GetByID(ctx, cmd.ServiceAgreementID)
		if err != nil {
			if errors.Is(err, ErrServiceAgreementNotFound) {
				return TimeEntryDTO{}, fmt.Errorf("record time entry: %w", ErrServiceAgreementNotFound)
			}
			return TimeEntryDTO{}, fmt.Errorf("record time entry: get service agreement: %w", err)
		}
		if !sa.Active {
			return TimeEntryDTO{}, fmt.Errorf("record time entry: %w", ErrInactiveServiceAgreement)
		}
	}

	hours, err := core.NewHours(cmd.Hours)
	if err != nil {
		return TimeEntryDTO{}, fmt.Errorf("record time entry: %w", err)
	}

	entry, err := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  cmd.CustomerProfileID,
		ServiceAgreementID: cmd.ServiceAgreementID,
		Description:        cmd.Description,
		Hours:              hours,
		Billable:           cmd.Billable,
		Date:               cmd.Date,
	})
	if err != nil {
		return TimeEntryDTO{}, fmt.Errorf("record time entry: %w", err)
	}

	if err := s.entries.Save(ctx, &entry); err != nil {
		return TimeEntryDTO{}, fmt.Errorf("record time entry: save: %w", err)
	}

	return timeEntryToDTO(entry), nil
}

// UpdateEntry fetches an existing entry, applies the update, and persists it.
func (s TimeEntryService) UpdateEntry(ctx context.Context, cmd UpdateTimeEntryCommand) (TimeEntryDTO, error) {
	entry, err := s.entries.GetByID(ctx, cmd.ID)
	if err != nil {
		if errors.Is(err, ErrTimeEntryNotFound) {
			return TimeEntryDTO{}, ErrTimeEntryNotFound
		}
		return TimeEntryDTO{}, fmt.Errorf("update time entry: get: %w", err)
	}

	hours, err := core.NewHours(cmd.Hours)
	if err != nil {
		return TimeEntryDTO{}, fmt.Errorf("update time entry: %w", err)
	}

	if err := entry.Update(cmd.Description, hours); err != nil {
		return TimeEntryDTO{}, fmt.Errorf("update time entry: %w", err)
	}

	if err := s.entries.Save(ctx, entry); err != nil {
		return TimeEntryDTO{}, fmt.Errorf("update time entry: save: %w", err)
	}

	return timeEntryToDTO(*entry), nil
}

// ListUnbilled returns all billable time entries with no assigned invoice for the given customer.
func (s TimeEntryService) ListUnbilled(ctx context.Context, customerID string) ([]TimeEntryDTO, error) {
	entries, err := s.entries.ListUnbilled(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list unbilled time entries: %w", err)
	}

	dtos := make([]TimeEntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, timeEntryToDTO(e))
	}
	return dtos, nil
}

// ListByCustomerProfile returns all time entries for the given customer profile, regardless of billing status.
func (s TimeEntryService) ListByCustomerProfile(ctx context.Context, customerID string) ([]TimeEntryDTO, error) {
	entries, err := s.entries.ListByCustomerProfile(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("list time entries by customer profile: %w", err)
	}

	dtos := make([]TimeEntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, timeEntryToDTO(e))
	}
	return dtos, nil
}

// Get fetches a single TimeEntry by ID and returns it as a DTO.
// Returns ErrTimeEntryNotFound if no entry matches.
func (s TimeEntryService) Get(ctx context.Context, id string) (TimeEntryDTO, error) {
	entry, err := s.entries.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrTimeEntryNotFound) {
			return TimeEntryDTO{}, ErrTimeEntryNotFound
		}
		return TimeEntryDTO{}, fmt.Errorf("get time entry: %w", err)
	}
	return timeEntryToDTO(*entry), nil
}

// DeleteEntry deletes a time entry by ID, provided it is not locked to an invoice.
// Returns ErrTimeEntryLocked if the entry is assigned to an invoice.
// Returns ErrTimeEntryNotFound if the entry does not exist.
func (s TimeEntryService) DeleteEntry(ctx context.Context, id string) error {
	entry, err := s.entries.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrTimeEntryNotFound) {
			return ErrTimeEntryNotFound
		}
		return fmt.Errorf("delete time entry: get: %w", err)
	}

	if entry.InvoiceID != "" {
		return fmt.Errorf("delete time entry: %w", core.ErrTimeEntryLocked)
	}

	if err := s.entries.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}

	return nil
}
