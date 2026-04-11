package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type timeEntryStoreStub struct {
	saveArg          *core.TimeEntry
	saveErr          error
	getByIDRes       *core.TimeEntry
	getByIDErr       error
	listUnbilledRes  []core.TimeEntry
	listUnbilledErr  error
	listByProfileRes []core.TimeEntry
	listByProfileErr error
	deleteTracker    func()
}

func (s *timeEntryStoreStub) Save(ctx context.Context, entry *core.TimeEntry) error {
	_ = ctx
	s.saveArg = entry
	return s.saveErr
}

func (s *timeEntryStoreStub) GetByID(ctx context.Context, id string) (*core.TimeEntry, error) {
	_ = ctx
	return s.getByIDRes, s.getByIDErr
}

func (s *timeEntryStoreStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	if s.deleteTracker != nil {
		s.deleteTracker()
	}
	return nil
}

func (s *timeEntryStoreStub) ListByCustomerProfile(ctx context.Context, customerID string) ([]core.TimeEntry, error) {
	_ = ctx
	return s.listByProfileRes, s.listByProfileErr
}

func (s *timeEntryStoreStub) ListUnbilled(ctx context.Context, customerID string) ([]core.TimeEntry, error) {
	_ = ctx
	return s.listUnbilledRes, s.listUnbilledErr
}

// customerProfileStoreForTimeEntry satisfies CustomerProfileStore.
type customerProfileStoreForTimeEntry struct {
	getByIDRes *core.CustomerProfile
	getByIDErr error
}

func (s *customerProfileStoreForTimeEntry) List(ctx context.Context, query ListQuery) (ListResult[core.CustomerProfile], error) {
	return ListResult[core.CustomerProfile]{}, nil
}
func (s *customerProfileStoreForTimeEntry) Save(ctx context.Context, p *core.CustomerProfile) error {
	return nil
}
func (s *customerProfileStoreForTimeEntry) GetByID(ctx context.Context, id string) (*core.CustomerProfile, error) {
	_ = ctx
	return s.getByIDRes, s.getByIDErr
}
func (s *customerProfileStoreForTimeEntry) Delete(ctx context.Context, id string) error {
	return nil
}

// serviceAgreementStoreForTimeEntry satisfies ServiceAgreementStore.
type serviceAgreementStoreForTimeEntry struct {
	getByIDRes *core.ServiceAgreement
	getByIDErr error
}

func (s *serviceAgreementStoreForTimeEntry) Save(ctx context.Context, sa *core.ServiceAgreement) error {
	return nil
}
func (s *serviceAgreementStoreForTimeEntry) GetByID(ctx context.Context, id string) (*core.ServiceAgreement, error) {
	_ = ctx
	return s.getByIDRes, s.getByIDErr
}
func (s *serviceAgreementStoreForTimeEntry) ListByCustomerProfileID(ctx context.Context, _ string) ([]core.ServiceAgreement, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func activeProfile() *core.CustomerProfile {
	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	return &core.CustomerProfile{
		ID:              "cus_abc123",
		LegalEntityID:   "le_001",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func activeAgreement() *core.ServiceAgreement {
	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	return &core.ServiceAgreement{
		ID:                "sa_xyz789",
		CustomerProfileID: "cus_abc123",
		Name:              "Support Agreement",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

func inactiveAgreement() *core.ServiceAgreement {
	a := activeAgreement()
	a.Active = false
	return a
}

// ---------------------------------------------------------------------------
// TimeEntryService.Record
// ---------------------------------------------------------------------------

func TestTimeEntryService_Record_NonExistentCustomer(t *testing.T) {
	t.Parallel()

	profiles := &customerProfileStoreForTimeEntry{getByIDErr: ErrCustomerProfileNotFound}
	entries := &timeEntryStoreStub{}
	svc := NewTimeEntryService(entries, profiles, nil)

	_, err := svc.Record(context.Background(), RecordTimeEntryCommand{
		CustomerProfileID:  "cus_nonexistent",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Work",
		Hours:              15000,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	if err == nil {
		t.Fatal("Record() error = nil, want error for non-existent customer")
	}
	if !errors.Is(err, ErrCustomerProfileNotFound) {
		t.Fatalf("error = %v, want wrapping ErrCustomerProfileNotFound", err)
	}
	if entries.saveArg != nil {
		t.Fatal("Save was unexpectedly called")
	}
}

func TestTimeEntryService_Record_InactiveAgreement(t *testing.T) {
	t.Parallel()

	profiles := &customerProfileStoreForTimeEntry{getByIDRes: activeProfile()}
	agreements := &serviceAgreementStoreForTimeEntry{getByIDRes: inactiveAgreement()}
	entries := &timeEntryStoreStub{}
	svc := NewTimeEntryService(entries, profiles, agreements)

	_, err := svc.Record(context.Background(), RecordTimeEntryCommand{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Work",
		Hours:              15000,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	if err == nil {
		t.Fatal("Record() error = nil, want error for inactive agreement")
	}
	if !errors.Is(err, ErrInactiveServiceAgreement) {
		t.Fatalf("error = %v, want wrapping ErrInactiveServiceAgreement", err)
	}
	if entries.saveArg != nil {
		t.Fatal("Save was unexpectedly called")
	}
}

func TestTimeEntryService_Record_ValidBillableEntry(t *testing.T) {
	t.Parallel()

	profiles := &customerProfileStoreForTimeEntry{getByIDRes: activeProfile()}
	agreements := &serviceAgreementStoreForTimeEntry{getByIDRes: activeAgreement()}
	entries := &timeEntryStoreStub{}
	svc := NewTimeEntryService(entries, profiles, agreements)

	workDate := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	dto, err := svc.Record(context.Background(), RecordTimeEntryCommand{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Initial implementation",
		Hours:              15000,
		Billable:           true,
		Date:               workDate,
	})

	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	if entries.saveArg == nil {
		t.Fatal("Save was not called")
	}
	if !strings.HasPrefix(dto.ID, "te_") {
		t.Fatalf("DTO ID = %q, want te_ prefix", dto.ID)
	}
	if dto.CustomerProfileID != "cus_abc123" {
		t.Fatalf("DTO CustomerProfileID = %q, want cus_abc123", dto.CustomerProfileID)
	}
	if dto.Hours != 15000 {
		t.Fatalf("DTO Hours = %d, want 15000", dto.Hours)
	}
	if !dto.Billable {
		t.Fatal("DTO Billable = false, want true")
	}
}

func TestTimeEntryService_Record_ValidNonBillableEntry(t *testing.T) {
	t.Parallel()

	profiles := &customerProfileStoreForTimeEntry{getByIDRes: activeProfile()}
	entries := &timeEntryStoreStub{}
	svc := NewTimeEntryService(entries, profiles, nil)

	dto, err := svc.Record(context.Background(), RecordTimeEntryCommand{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Internal meeting",
		Hours:              10000,
		Billable:           false,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	if err != nil {
		t.Fatalf("Record() error = %v for non-billable", err)
	}
	if dto.Billable {
		t.Fatal("DTO Billable = true, want false")
	}
	if dto.ServiceAgreementID != "sa_xyz789" {
		t.Fatalf("DTO ServiceAgreementID = %q, want sa_xyz789", dto.ServiceAgreementID)
	}
}

func TestTimeEntryService_Record_RejectsMissingServiceAgreement(t *testing.T) {
	t.Parallel()

	profiles := &customerProfileStoreForTimeEntry{getByIDRes: activeProfile()}
	entries := &timeEntryStoreStub{}
	svc := NewTimeEntryService(entries, profiles, nil)

	_, err := svc.Record(context.Background(), RecordTimeEntryCommand{
		CustomerProfileID: "cus_abc123",
		Description:       "Internal meeting",
		Hours:             10000,
		Billable:          false,
		Date:              time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	if err == nil {
		t.Fatal("Record() error = nil, want error for missing service agreement")
	}
	if !errors.Is(err, core.ErrMissingServiceAgreement) {
		t.Fatalf("error = %v, want wrapping core.ErrMissingServiceAgreement", err)
	}
	if entries.saveArg != nil {
		t.Fatal("Save was unexpectedly called")
	}
}

// ---------------------------------------------------------------------------
// TimeEntryService.UpdateEntry
// ---------------------------------------------------------------------------

func TestTimeEntryService_UpdateEntry_LockedEntry(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	lockedEntry, _ := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Work",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	_ = lockedEntry.AssignToInvoice("inv_001")

	entries := &timeEntryStoreStub{getByIDRes: &lockedEntry}
	svc := NewTimeEntryService(entries, nil, nil)

	_, err := svc.UpdateEntry(context.Background(), UpdateTimeEntryCommand{
		ID:          lockedEntry.ID,
		Description: "Updated",
		Hours:       20000,
	})

	if err == nil {
		t.Fatal("UpdateEntry() error = nil, want locked error")
	}
	if !errors.Is(err, core.ErrTimeEntryLocked) {
		t.Fatalf("error = %v, want wrapping core.ErrTimeEntryLocked", err)
	}
}

func TestTimeEntryService_UpdateEntry_ValidUpdate(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	existingEntry, _ := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	entries := &timeEntryStoreStub{getByIDRes: &existingEntry}
	svc := NewTimeEntryService(entries, nil, nil)

	dto, err := svc.UpdateEntry(context.Background(), UpdateTimeEntryCommand{
		ID:          existingEntry.ID,
		Description: "Updated description",
		Hours:       20000,
	})

	if err != nil {
		t.Fatalf("UpdateEntry() error = %v", err)
	}
	if entries.saveArg == nil {
		t.Fatal("Save was not called")
	}
	if dto.Description != "Updated description" {
		t.Fatalf("DTO Description = %q, want Updated description", dto.Description)
	}
	if dto.Hours != 20000 {
		t.Fatalf("DTO Hours = %d, want 20000", dto.Hours)
	}
}

// ---------------------------------------------------------------------------
// TimeEntryService.ListUnbilled
// ---------------------------------------------------------------------------

func TestTimeEntryService_ListUnbilled_ReturnsBillableUnassigned(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(15000)

	unbilledEntries := []core.TimeEntry{
		{
			ID:                "te_001",
			CustomerProfileID: "cus_abc123",
			Description:       "Work A",
			Hours:             hours,
			Billable:          true,
			InvoiceID:         "",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "te_002",
			CustomerProfileID: "cus_abc123",
			Description:       "Work B",
			Hours:             hours,
			Billable:          true,
			InvoiceID:         "",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	entries := &timeEntryStoreStub{listUnbilledRes: unbilledEntries}
	svc := NewTimeEntryService(entries, nil, nil)

	dtos, err := svc.ListUnbilled(context.Background(), "cus_abc123")
	if err != nil {
		t.Fatalf("ListUnbilled() error = %v", err)
	}
	if len(dtos) != 2 {
		t.Fatalf("len(dtos) = %d, want 2", len(dtos))
	}
	if dtos[0].ID != "te_001" {
		t.Fatalf("dtos[0].ID = %q, want te_001", dtos[0].ID)
	}
	if dtos[1].ID != "te_002" {
		t.Fatalf("dtos[1].ID = %q, want te_002", dtos[1].ID)
	}
}

func TestTimeEntryService_ListUnbilled_EmptyWhenAllInvoiced(t *testing.T) {
	t.Parallel()

	// Store returns empty — simulates all entries already invoiced (filtered by store)
	entries := &timeEntryStoreStub{listUnbilledRes: []core.TimeEntry{}}
	svc := NewTimeEntryService(entries, nil, nil)

	dtos, err := svc.ListUnbilled(context.Background(), "cus_abc123")
	if err != nil {
		t.Fatalf("ListUnbilled() error = %v", err)
	}
	if len(dtos) != 0 {
		t.Fatalf("len(dtos) = %d, want 0 when all entries are invoiced", len(dtos))
	}
}

// ---------------------------------------------------------------------------
// TimeEntryService.UpdateEntry — invariant re-validation via app layer
// ---------------------------------------------------------------------------

func TestTimeEntryService_UpdateEntry_RejectsBlankDescription(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	existingEntry, _ := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	entries := &timeEntryStoreStub{getByIDRes: &existingEntry}
	svc := NewTimeEntryService(entries, nil, nil)

	_, err := svc.UpdateEntry(context.Background(), UpdateTimeEntryCommand{
		ID:          existingEntry.ID,
		Description: "",
		Hours:       15000,
	})

	if err == nil {
		t.Fatal("UpdateEntry() error = nil, want error for blank description")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "description") {
		t.Fatalf("error = %q, want contains 'description'", err.Error())
	}
	if entries.saveArg != nil {
		t.Fatal("Save was unexpectedly called after rejected blank description")
	}
}

func TestTimeEntryService_UpdateEntry_RejectsNonPositiveHours(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	existingEntry, _ := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Original",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	entries := &timeEntryStoreStub{getByIDRes: &existingEntry}
	svc := NewTimeEntryService(entries, nil, nil)

	_, err := svc.UpdateEntry(context.Background(), UpdateTimeEntryCommand{
		ID:          existingEntry.ID,
		Description: "Updated",
		Hours:       0, // non-positive
	})

	if err == nil {
		t.Fatal("UpdateEntry() error = nil, want error for non-positive hours")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "hours") {
		t.Fatalf("error = %q, want contains 'hours'", err.Error())
	}
	if entries.saveArg != nil {
		t.Fatal("Save was unexpectedly called after rejected non-positive hours")
	}
}

// ---------------------------------------------------------------------------
// TimeEntryService.ListByCustomerProfile
// ---------------------------------------------------------------------------

func TestTimeEntryService_ListByCustomerProfile_ReturnsAllEntries(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	hours, _ := core.NewHours(15000)

	allEntries := []core.TimeEntry{
		{
			ID:                "te_001",
			CustomerProfileID: "cus_abc123",
			Description:       "Work A",
			Hours:             hours,
			Billable:          true,
			InvoiceID:         "inv_001", // already invoiced
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "te_002",
			CustomerProfileID: "cus_abc123",
			Description:       "Work B",
			Hours:             hours,
			Billable:          false, // non-billable
			InvoiceID:         "",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "te_003",
			CustomerProfileID: "cus_abc123",
			Description:       "Work C",
			Hours:             hours,
			Billable:          true,
			InvoiceID:         "",
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	entries := &timeEntryStoreStub{listByProfileRes: allEntries}
	svc := NewTimeEntryService(entries, nil, nil)

	dtos, err := svc.ListByCustomerProfile(context.Background(), "cus_abc123")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	// Returns ALL entries (invoiced, non-billable, unbilled) — no filtering here
	if len(dtos) != 3 {
		t.Fatalf("len(dtos) = %d, want 3", len(dtos))
	}
	if dtos[0].ID != "te_001" {
		t.Fatalf("dtos[0].ID = %q, want te_001", dtos[0].ID)
	}
	if dtos[2].ID != "te_003" {
		t.Fatalf("dtos[2].ID = %q, want te_003", dtos[2].ID)
	}
}

func TestTimeEntryService_ListByCustomerProfile_EmptyForNoEntries(t *testing.T) {
	t.Parallel()

	entries := &timeEntryStoreStub{listByProfileRes: []core.TimeEntry{}}
	svc := NewTimeEntryService(entries, nil, nil)

	dtos, err := svc.ListByCustomerProfile(context.Background(), "cus_abc123")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(dtos) != 0 {
		t.Fatalf("len(dtos) = %d, want 0 for customer with no entries", len(dtos))
	}
}

// ---------------------------------------------------------------------------
// TimeEntryService.DeleteEntry
// ---------------------------------------------------------------------------

func TestTimeEntryService_DeleteEntry_DeletesDraftEntry(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	draftEntry, _ := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Draft work",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})

	deleteCount := 0
	entries := &timeEntryStoreStub{getByIDRes: &draftEntry}
	entries.deleteTracker = func() { deleteCount++ }
	svc := NewTimeEntryService(entries, nil, nil)

	err := svc.DeleteEntry(context.Background(), draftEntry.ID)
	if err != nil {
		t.Fatalf("DeleteEntry() error = %v, want nil for draft entry", err)
	}
	if deleteCount != 1 {
		t.Fatalf("Delete called %d times, want 1", deleteCount)
	}
}

func TestTimeEntryService_DeleteEntry_RejectsLockedEntry(t *testing.T) {
	t.Parallel()

	hours, _ := core.NewHours(15000)
	lockedEntry, _ := core.NewTimeEntry(core.TimeEntryParams{
		CustomerProfileID:  "cus_abc123",
		ServiceAgreementID: "sa_xyz789",
		Description:        "Invoiced work",
		Hours:              hours,
		Billable:           true,
		Date:               time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC),
	})
	_ = lockedEntry.AssignToInvoice("inv_001")

	deleteCount := 0
	entries := &timeEntryStoreStub{getByIDRes: &lockedEntry}
	entries.deleteTracker = func() { deleteCount++ }
	svc := NewTimeEntryService(entries, nil, nil)

	err := svc.DeleteEntry(context.Background(), lockedEntry.ID)
	if err == nil {
		t.Fatal("DeleteEntry() error = nil, want locked error for invoiced entry")
	}
	if !errors.Is(err, core.ErrTimeEntryLocked) {
		t.Fatalf("error = %v, want wrapping core.ErrTimeEntryLocked", err)
	}
	if deleteCount != 0 {
		t.Fatal("Delete was unexpectedly called on a locked entry")
	}
}
