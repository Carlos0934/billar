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

type serviceAgreementStoreStub struct {
	saveArg                    *core.ServiceAgreement
	saveErr                    error
	getByIDArg                 string
	getByIDRes                 *core.ServiceAgreement
	getByIDErr                 error
	listByCustomerProfileIDArg string
	listByCustomerProfileIDRes []core.ServiceAgreement
	listByCustomerProfileIDErr error
}

func (s *serviceAgreementStoreStub) Save(ctx context.Context, sa *core.ServiceAgreement) error {
	_ = ctx
	s.saveArg = sa
	return s.saveErr
}

func (s *serviceAgreementStoreStub) GetByID(ctx context.Context, id string) (*core.ServiceAgreement, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *serviceAgreementStoreStub) ListByCustomerProfileID(ctx context.Context, customerProfileID string) ([]core.ServiceAgreement, error) {
	_ = ctx
	s.listByCustomerProfileIDArg = customerProfileID
	return s.listByCustomerProfileIDRes, s.listByCustomerProfileIDErr
}

// customerProfileStoreForAgreements is a minimal stub satisfying CustomerProfileStore.
type customerProfileStoreForAgreements struct {
	getByIDArg string
	getByIDRes *core.CustomerProfile
	getByIDErr error
}

func (s *customerProfileStoreForAgreements) List(ctx context.Context, query ListQuery) (ListResult[core.CustomerProfile], error) {
	return ListResult[core.CustomerProfile]{}, nil
}

func (s *customerProfileStoreForAgreements) Save(ctx context.Context, profile *core.CustomerProfile) error {
	return nil
}

func (s *customerProfileStoreForAgreements) GetByID(ctx context.Context, id string) (*core.CustomerProfile, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *customerProfileStoreForAgreements) Delete(ctx context.Context, id string) error {
	return nil
}

// ---------------------------------------------------------------------------
// AgreementService.Create
// ---------------------------------------------------------------------------

func TestAgreementService_Create_UnknownCustomer(t *testing.T) {
	t.Parallel()

	agreements := &serviceAgreementStoreStub{}
	profiles := &customerProfileStoreForAgreements{
		getByIDErr: ErrCustomerProfileNotFound,
	}
	svc := NewAgreementService(agreements, profiles)

	_, err := svc.Create(context.Background(), CreateServiceAgreementCommand{
		CustomerProfileID: "cus_nonexistent",
		Name:              "Support",
		BillingMode:       "hourly",
		HourlyRate:        1000,
		Currency:          "USD",
	})

	if err == nil {
		t.Fatal("Create() error = nil, want non-nil for unknown customer")
	}
	if agreements.saveArg != nil {
		t.Fatal("Save was unexpectedly called")
	}
}

func TestAgreementService_Create_ValidCustomer(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	activeProfile := &core.CustomerProfile{
		ID:              "cus_abc123",
		LegalEntityID:   "le_456",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	agreements := &serviceAgreementStoreStub{}
	profiles := &customerProfileStoreForAgreements{getByIDRes: activeProfile}
	svc := NewAgreementService(agreements, profiles)

	dto, err := svc.Create(context.Background(), CreateServiceAgreementCommand{
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		BillingMode:       "hourly",
		HourlyRate:        1000,
		Currency:          "USD",
	})

	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if agreements.saveArg == nil {
		t.Fatal("Save not called")
	}
	if dto.ID == "" {
		t.Fatal("returned DTO ID is empty")
	}
	if !strings.HasPrefix(dto.ID, "sa_") {
		t.Fatalf("DTO ID = %q, want sa_ prefix", dto.ID)
	}
	if dto.CustomerProfileID != "cus_abc123" {
		t.Fatalf("DTO CustomerProfileID = %q, want cus_abc123", dto.CustomerProfileID)
	}
	if dto.HourlyRate != 1000 {
		t.Fatalf("DTO HourlyRate = %d, want 1000", dto.HourlyRate)
	}
	if !dto.Active {
		t.Fatal("DTO Active = false, want true")
	}
}

// ---------------------------------------------------------------------------
// AgreementService.UpdateRate
// ---------------------------------------------------------------------------

func TestAgreementService_UpdateRate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existingAgreement := &core.ServiceAgreement{
		ID:                "sa_existing",
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	tests := []struct {
		name      string
		id        string
		cmd       UpdateServiceAgreementRateCommand
		storeRes  *core.ServiceAgreement
		storeErr  error
		wantErr   string
		wantSaved bool
		wantRate  int64
	}{
		{
			name:      "updates rate and saves",
			id:        "sa_existing",
			cmd:       UpdateServiceAgreementRateCommand{HourlyRate: 1500},
			storeRes:  existingAgreement,
			wantSaved: true,
			wantRate:  1500,
		},
		{
			name:     "not found returns error",
			id:       "sa_nonexistent",
			cmd:      UpdateServiceAgreementRateCommand{HourlyRate: 1500},
			storeErr: ErrServiceAgreementNotFound,
			wantErr:  "not found",
		},
		{
			name:     "zero rate returns error",
			id:       "sa_existing",
			cmd:      UpdateServiceAgreementRateCommand{HourlyRate: 0},
			storeRes: existingAgreement,
			wantErr:  "hourly rate",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			saStore := &serviceAgreementStoreStub{
				getByIDRes: tc.storeRes,
				getByIDErr: tc.storeErr,
			}
			svc := NewAgreementService(saStore, nil)

			dto, err := svc.UpdateRate(context.Background(), tc.id, tc.cmd)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("UpdateRate() error = nil, want non-nil")
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantErr) {
					t.Fatalf("error = %q, want contains %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("UpdateRate() error = %v", err)
			}
			if tc.wantSaved && saStore.saveArg == nil {
				t.Fatal("Save not called")
			}
			if dto.HourlyRate != tc.wantRate {
				t.Fatalf("DTO HourlyRate = %d, want %d", dto.HourlyRate, tc.wantRate)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AgreementService.Activate / Deactivate
// ---------------------------------------------------------------------------

func TestAgreementService_Activate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	inactiveAgreement := &core.ServiceAgreement{
		ID:                "sa_existing",
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            false,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	saStore := &serviceAgreementStoreStub{getByIDRes: inactiveAgreement}
	svc := NewAgreementService(saStore, nil)

	dto, err := svc.Activate(context.Background(), "sa_existing")
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if !dto.Active {
		t.Fatal("DTO Active = false after Activate(), want true")
	}
	if saStore.saveArg == nil {
		t.Fatal("Save not called")
	}
}

func TestAgreementService_Deactivate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	activeAgreement := &core.ServiceAgreement{
		ID:                "sa_existing",
		CustomerProfileID: "cus_abc123",
		Name:              "Monthly Support",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1000,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	saStore := &serviceAgreementStoreStub{getByIDRes: activeAgreement}
	svc := NewAgreementService(saStore, nil)

	dto, err := svc.Deactivate(context.Background(), "sa_existing")
	if err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}
	if dto.Active {
		t.Fatal("DTO Active = true after Deactivate(), want false")
	}
	if saStore.saveArg == nil {
		t.Fatal("Save not called")
	}
}

func TestAgreementService_Activate_NotFound(t *testing.T) {
	t.Parallel()

	saStore := &serviceAgreementStoreStub{getByIDErr: ErrServiceAgreementNotFound}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.Activate(context.Background(), "sa_nonexistent")
	if err == nil {
		t.Fatal("Activate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %q, want contains 'not found'", err.Error())
	}
}

func TestAgreementService_Deactivate_NotFound(t *testing.T) {
	t.Parallel()

	saStore := &serviceAgreementStoreStub{getByIDErr: ErrServiceAgreementNotFound}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.Deactivate(context.Background(), "sa_nonexistent")
	if err == nil {
		t.Fatal("Deactivate() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %q, want contains 'not found'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// AgreementService.ListByCustomerProfile
// ---------------------------------------------------------------------------

func TestAgreementService_ListByCustomerProfile(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	agreements := []core.ServiceAgreement{
		{
			ID:                "sa_1",
			CustomerProfileID: "cus_abc123",
			Name:              "Support A",
			BillingMode:       core.BillingModeHourly,
			HourlyRate:        1000,
			Currency:          "USD",
			Active:            true,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			ID:                "sa_2",
			CustomerProfileID: "cus_abc123",
			Name:              "Support B",
			BillingMode:       core.BillingModeHourly,
			HourlyRate:        2000,
			Currency:          "EUR",
			Active:            false,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	saStore := &serviceAgreementStoreStub{listByCustomerProfileIDRes: agreements}
	svc := NewAgreementService(saStore, nil)

	dtos, err := svc.ListByCustomerProfile(context.Background(), "cus_abc123")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(dtos) != 2 {
		t.Fatalf("len(dtos) = %d, want 2", len(dtos))
	}
	if dtos[0].ID != "sa_1" {
		t.Fatalf("dtos[0].ID = %q, want sa_1", dtos[0].ID)
	}
	if dtos[1].HourlyRate != 2000 {
		t.Fatalf("dtos[1].HourlyRate = %d, want 2000", dtos[1].HourlyRate)
	}
	if saStore.listByCustomerProfileIDArg != "cus_abc123" {
		t.Fatalf("ListByCustomerProfileID called with %q, want cus_abc123", saStore.listByCustomerProfileIDArg)
	}
}

func TestAgreementService_ListByCustomerProfile_Empty(t *testing.T) {
	t.Parallel()

	// Stub returns empty slice; confirms real filtering, not trivial nil
	saStore := &serviceAgreementStoreStub{listByCustomerProfileIDRes: []core.ServiceAgreement{}}
	svc := NewAgreementService(saStore, nil)

	dtos, err := svc.ListByCustomerProfile(context.Background(), "cus_no_agreements")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(dtos) != 0 {
		t.Fatalf("expected 0 dtos, got %d", len(dtos))
	}
}

// ---------------------------------------------------------------------------
// AgreementService.Create — additional error branches
// ---------------------------------------------------------------------------

func TestAgreementService_Create_ProfileStoreUnexpectedError(t *testing.T) {
	t.Parallel()

	unexpectedErr := errors.New("db connection lost")
	agreements := &serviceAgreementStoreStub{}
	profiles := &customerProfileStoreForAgreements{getByIDErr: unexpectedErr}
	svc := NewAgreementService(agreements, profiles)

	_, err := svc.Create(context.Background(), CreateServiceAgreementCommand{
		CustomerProfileID: "cus_abc123",
		Name:              "Support",
		BillingMode:       "hourly",
		HourlyRate:        1000,
		Currency:          "USD",
	})

	if err == nil {
		t.Fatal("Create() error = nil, want non-nil for unexpected profile store error")
	}
	if agreements.saveArg != nil {
		t.Fatal("Save was unexpectedly called")
	}
}

func TestAgreementService_Create_SaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("disk full")
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	activeProfile := &core.CustomerProfile{
		ID:              "cus_abc123",
		LegalEntityID:   "le_456",
		Status:          core.CustomerProfileStatusActive,
		DefaultCurrency: "USD",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	agreements := &serviceAgreementStoreStub{saveErr: saveErr}
	profiles := &customerProfileStoreForAgreements{getByIDRes: activeProfile}
	svc := NewAgreementService(agreements, profiles)

	_, err := svc.Create(context.Background(), CreateServiceAgreementCommand{
		CustomerProfileID: "cus_abc123",
		Name:              "Support",
		BillingMode:       "hourly",
		HourlyRate:        1000,
		Currency:          "USD",
	})

	if err == nil {
		t.Fatal("Create() error = nil, want non-nil when Save fails")
	}
}

// ---------------------------------------------------------------------------
// AgreementService.UpdateRate — store error on GetByID (non-NotFound)
// ---------------------------------------------------------------------------

func TestAgreementService_UpdateRate_StoreError(t *testing.T) {
	t.Parallel()

	unexpectedErr := errors.New("db timeout")
	saStore := &serviceAgreementStoreStub{getByIDErr: unexpectedErr}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.UpdateRate(context.Background(), "sa_x", UpdateServiceAgreementRateCommand{HourlyRate: 500})
	if err == nil {
		t.Fatal("UpdateRate() error = nil, want non-nil for unexpected store error")
	}
}

// ---------------------------------------------------------------------------
// AgreementService.Activate / Deactivate — Save error branch
// ---------------------------------------------------------------------------

func TestAgreementService_Activate_SaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("write failed")
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	saStore := &serviceAgreementStoreStub{
		getByIDRes: &core.ServiceAgreement{ID: "sa_x", Active: false, CreatedAt: now, UpdatedAt: now},
		saveErr:    saveErr,
	}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.Activate(context.Background(), "sa_x")
	if err == nil {
		t.Fatal("Activate() error = nil, want non-nil when Save fails")
	}
}

func TestAgreementService_Deactivate_SaveError(t *testing.T) {
	t.Parallel()

	saveErr := errors.New("write failed")
	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	saStore := &serviceAgreementStoreStub{
		getByIDRes: &core.ServiceAgreement{ID: "sa_x", Active: true, CreatedAt: now, UpdatedAt: now},
		saveErr:    saveErr,
	}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.Deactivate(context.Background(), "sa_x")
	if err == nil {
		t.Fatal("Deactivate() error = nil, want non-nil when Save fails")
	}
}

// ---------------------------------------------------------------------------
// AgreementService.Activate — store error on GetByID (non-NotFound)
// ---------------------------------------------------------------------------

func TestAgreementService_Activate_StoreError(t *testing.T) {
	t.Parallel()

	unexpectedErr := errors.New("db timeout")
	saStore := &serviceAgreementStoreStub{getByIDErr: unexpectedErr}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.Activate(context.Background(), "sa_x")
	if err == nil {
		t.Fatal("Activate() error = nil, want non-nil for unexpected store error")
	}
}

func TestAgreementService_Deactivate_StoreError(t *testing.T) {
	t.Parallel()

	unexpectedErr := errors.New("db timeout")
	saStore := &serviceAgreementStoreStub{getByIDErr: unexpectedErr}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.Deactivate(context.Background(), "sa_x")
	if err == nil {
		t.Fatal("Deactivate() error = nil, want non-nil for unexpected store error")
	}
}

// ---------------------------------------------------------------------------
// AgreementService.Get
// ---------------------------------------------------------------------------

func TestAgreementService_Get(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	existingAgreement := &core.ServiceAgreement{
		ID:                "sa_get_test",
		CustomerProfileID: "cus_abc123",
		Name:              "Get Test Agreement",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        1200,
		Currency:          "USD",
		Active:            true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	tests := []struct {
		name     string
		id       string
		storeRes *core.ServiceAgreement
		storeErr error
		wantErr  string
		wantID   string
		wantRate int64
	}{
		{
			name:     "returns agreement dto when found",
			id:       "sa_get_test",
			storeRes: existingAgreement,
			wantID:   "sa_get_test",
			wantRate: 1200,
		},
		{
			name:     "returns not found error when agreement does not exist",
			id:       "sa_nonexistent",
			storeErr: ErrServiceAgreementNotFound,
			wantErr:  "not found",
		},
		{
			name:     "propagates unexpected store error",
			id:       "sa_err",
			storeErr: errors.New("db timeout"),
			wantErr:  "db timeout",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			saStore := &serviceAgreementStoreStub{
				getByIDRes: tc.storeRes,
				getByIDErr: tc.storeErr,
			}
			svc := NewAgreementService(saStore, nil)

			dto, err := svc.Get(context.Background(), tc.id)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Get() error = nil, want non-nil")
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantErr) {
					t.Fatalf("error = %q, want contains %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if dto.ID != tc.wantID {
				t.Fatalf("DTO ID = %q, want %q", dto.ID, tc.wantID)
			}
			if dto.HourlyRate != tc.wantRate {
				t.Fatalf("DTO HourlyRate = %d, want %d", dto.HourlyRate, tc.wantRate)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AgreementService.ListByCustomerProfile — store error branch
// ---------------------------------------------------------------------------

func TestAgreementService_ListByCustomerProfile_StoreError(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("read failed")
	saStore := &serviceAgreementStoreStub{listByCustomerProfileIDErr: storeErr}
	svc := NewAgreementService(saStore, nil)

	_, err := svc.ListByCustomerProfile(context.Background(), "cus_abc123")
	if err == nil {
		t.Fatal("ListByCustomerProfile() error = nil, want non-nil for store error")
	}
}

// ---------------------------------------------------------------------------
// serviceAgreementToDTO / formatServiceAgreementTime — DTO branch coverage
// ---------------------------------------------------------------------------

func TestServiceAgreementToDTO_WithValidDates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	future := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	sa := core.ServiceAgreement{
		ID:                "sa_test",
		CustomerProfileID: "cus_abc123",
		Name:              "Test Agreement",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        500,
		Currency:          "USD",
		Active:            true,
		ValidFrom:         &now,
		ValidUntil:        &future,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	dto := serviceAgreementToDTO(sa)

	if dto.ValidFrom == nil {
		t.Fatal("ValidFrom = nil, want non-nil when set")
	}
	if *dto.ValidFrom != "2026-04-05T12:00:00Z" {
		t.Fatalf("ValidFrom = %q, want RFC3339 formatted", *dto.ValidFrom)
	}
	if dto.ValidUntil == nil {
		t.Fatal("ValidUntil = nil, want non-nil when set")
	}
	if *dto.ValidUntil != "2027-01-01T00:00:00Z" {
		t.Fatalf("ValidUntil = %q, want RFC3339 formatted", *dto.ValidUntil)
	}
	if dto.CreatedAt != "2026-04-05T12:00:00Z" {
		t.Fatalf("CreatedAt = %q, want RFC3339 formatted", dto.CreatedAt)
	}
}

func TestServiceAgreementToDTO_NilDates(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	sa := core.ServiceAgreement{
		ID:                "sa_test2",
		CustomerProfileID: "cus_abc123",
		Name:              "No Dates",
		BillingMode:       core.BillingModeHourly,
		HourlyRate:        500,
		Currency:          "USD",
		Active:            true,
		ValidFrom:         nil,
		ValidUntil:        nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	dto := serviceAgreementToDTO(sa)

	if dto.ValidFrom != nil {
		t.Fatalf("ValidFrom = %v, want nil when not set", dto.ValidFrom)
	}
	if dto.ValidUntil != nil {
		t.Fatalf("ValidUntil = %v, want nil when not set", dto.ValidUntil)
	}
}

func TestFormatServiceAgreementTime_Zero(t *testing.T) {
	t.Parallel()

	result := formatServiceAgreementTime(time.Time{})
	if result != "" {
		t.Fatalf("formatServiceAgreementTime(zero) = %q, want empty string", result)
	}
}

func TestFormatServiceAgreementTime_NonZero(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	result := formatServiceAgreementTime(ts)
	if result != "2026-04-05T12:00:00Z" {
		t.Fatalf("formatServiceAgreementTime() = %q, want RFC3339 formatted", result)
	}
}
