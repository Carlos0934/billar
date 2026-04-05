package app

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type customerProfileStoreStub struct {
	called bool
	query  ListQuery
	result ListResult[core.CustomerProfile]
	err    error
	// Write operations
	saveArg    *core.CustomerProfile
	saveErr    error
	getByIDArg string
	getByIDRes *core.CustomerProfile
	getByIDErr error
	deleteArg  string
	deleteErr  error
}

func (s *customerProfileStoreStub) List(ctx context.Context, query ListQuery) (ListResult[core.CustomerProfile], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func (s *customerProfileStoreStub) Save(ctx context.Context, profile *core.CustomerProfile) error {
	_ = ctx
	s.saveArg = profile
	return s.saveErr
}

func (s *customerProfileStoreStub) GetByID(ctx context.Context, id string) (*core.CustomerProfile, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *customerProfileStoreStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteArg = id
	return s.deleteErr
}

type legalEntityStoreStubForCustomer struct {
	getByIDArg string
	getByIDRes *core.LegalEntity
	getByIDErr error
}

func (s *legalEntityStoreStubForCustomer) List(ctx context.Context, query ListQuery) (ListResult[core.LegalEntity], error) {
	return ListResult[core.LegalEntity]{}, nil
}

func (s *legalEntityStoreStubForCustomer) Save(ctx context.Context, entity *core.LegalEntity) error {
	return nil
}

func (s *legalEntityStoreStubForCustomer) GetByID(ctx context.Context, id string) (*core.LegalEntity, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *legalEntityStoreStubForCustomer) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCustomerProfileService_List(t *testing.T) {
	t.Parallel()

	createdAt := "2026-04-03T10:00:00Z"
	updatedAt := "2026-04-03T10:05:00Z"

	tests := []struct {
		name         string
		query        ListQuery
		storeResult  ListResult[core.CustomerProfile]
		storeErr     error
		wantQuery    ListQuery
		wantResult   ListResult[CustomerProfileDTO]
		wantErr      string
		wantStoreHit bool
	}{
		{
			name: "returns mapped list with normalized query",
			query: ListQuery{
				Page:      0,
				PageSize:  500,
				Search:    "  Acme  ",
				SortField: " name ",
				SortDir:   " DESC ",
			},
			storeResult: ListResult[core.CustomerProfile]{
				Items: []core.CustomerProfile{{
					ID:              "cus_123",
					LegalEntityID:   "le_456",
					Status:          core.CustomerProfileStatusActive,
					DefaultCurrency: "USD",
					Notes:           "Preferred customer",
					CreatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
					UpdatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
				}},
				Total:    1,
				Page:     1,
				PageSize: 100,
			},
			wantQuery: ListQuery{Page: 1, PageSize: 100, Search: "Acme", SortField: "legal_name", SortDir: "desc"},
			wantResult: ListResult[CustomerProfileDTO]{
				Items: []CustomerProfileDTO{{
					ID:              "cus_123",
					LegalEntityID:   "le_456",
					Status:          string(core.CustomerProfileStatusActive),
					DefaultCurrency: "USD",
					Notes:           "Preferred customer",
					CreatedAt:       createdAt,
					UpdatedAt:       updatedAt,
				}},
				Total:    1,
				Page:     1,
				PageSize: 100,
			},
			wantStoreHit: true,
		},
		{
			name:         "propagates store errors",
			query:        ListQuery{Page: 1, PageSize: 20},
			storeResult:  ListResult[core.CustomerProfile]{},
			storeErr:     errors.New("database error"),
			wantErr:      "database error",
			wantStoreHit: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &customerProfileStoreStub{result: tc.storeResult, err: tc.storeErr}
			svc := NewCustomerProfileService(nil, store)

			got, err := svc.List(context.Background(), tc.query)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("List() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("List() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				if store.called != tc.wantStoreHit {
					t.Fatalf("store called = %v, want %v", store.called, tc.wantStoreHit)
				}
				return
			}

			if err != nil {
				t.Fatalf("List() error = %v", err)
			}
			if store.query != tc.wantQuery {
				t.Fatalf("store query = %+v, want %+v", store.query, tc.wantQuery)
			}
			if !reflect.DeepEqual(got, tc.wantResult) {
				t.Fatalf("List() = %+v, want %+v", got, tc.wantResult)
			}
		})
	}
}

func TestCustomerProfileService_Create(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		cmd               CreateCustomerProfileCommand
		legalEntityStore  *legalEntityStoreStubForCustomer
		profileStore      *customerProfileStoreStub
		wantErr           string
		wantSaved         bool
		savedFields       core.CustomerProfile
		wantLegalEntityID string
	}{
		{
			name: "creates customer profile with valid legal entity",
			cmd: CreateCustomerProfileCommand{
				LegalEntityID:   "le_123",
				DefaultCurrency: "USD",
				Notes:           "Preferred customer",
			},
			legalEntityStore: &legalEntityStoreStubForCustomer{
				getByIDRes: &core.LegalEntity{
					ID:        "le_123",
					Type:      core.EntityTypeCompany,
					LegalName: "Acme SRL",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			profileStore: &customerProfileStoreStub{},
			wantSaved:    true,
			savedFields: core.CustomerProfile{
				LegalEntityID:   "le_123",
				DefaultCurrency: "USD",
				Notes:           "Preferred customer",
				Status:          core.CustomerProfileStatusActive,
			},
			wantLegalEntityID: "le_123",
		},
		{
			name: "rejects non-existent legal entity",
			cmd: CreateCustomerProfileCommand{
				LegalEntityID:   "le_nonexistent",
				DefaultCurrency: "USD",
			},
			legalEntityStore: &legalEntityStoreStubForCustomer{
				getByIDErr: ErrLegalEntityNotFound,
			},
			profileStore: &customerProfileStoreStub{},
			wantErr:      "legal entity not found",
		},
		{
			name: "rejects missing legal entity id",
			cmd: CreateCustomerProfileCommand{
				DefaultCurrency: "USD",
			},
			legalEntityStore: &legalEntityStoreStubForCustomer{},
			profileStore:     &customerProfileStoreStub{},
			wantErr:          "legal entity id is required",
		},
		{
			name: "rejects missing default currency",
			cmd: CreateCustomerProfileCommand{
				LegalEntityID: "le_123",
			},
			legalEntityStore: &legalEntityStoreStubForCustomer{},
			profileStore:     &customerProfileStoreStub{},
			wantErr:          "default currency is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewCustomerProfileService(tc.legalEntityStore, tc.profileStore)

			got, err := svc.Create(context.Background(), tc.cmd)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Create() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Create() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				if tc.profileStore.saveArg != nil {
					t.Fatalf("profile store.Save called unexpectedly, arg = %+v", tc.profileStore.saveArg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if tc.wantSaved && tc.profileStore.saveArg == nil {
				t.Fatal("profile store.Save not called")
			}

			if tc.wantSaved {
				if tc.profileStore.saveArg.LegalEntityID != tc.savedFields.LegalEntityID {
					t.Errorf("saved LegalEntityID = %v, want %v", tc.profileStore.saveArg.LegalEntityID, tc.savedFields.LegalEntityID)
				}
				if tc.profileStore.saveArg.DefaultCurrency != tc.savedFields.DefaultCurrency {
					t.Errorf("saved DefaultCurrency = %v, want %v", tc.profileStore.saveArg.DefaultCurrency, tc.savedFields.DefaultCurrency)
				}
				if tc.profileStore.saveArg.Status != tc.savedFields.Status {
					t.Errorf("saved Status = %v, want %v", tc.profileStore.saveArg.Status, tc.savedFields.Status)
				}
				if tc.profileStore.saveArg.ID == "" {
					t.Error("saved customer profile ID is empty")
				}
				if got.ID == "" {
					t.Error("returned customer profile ID is empty")
				}
			}

			if tc.wantLegalEntityID != "" && tc.legalEntityStore.getByIDArg != tc.wantLegalEntityID {
				t.Errorf("legal entity store.GetByID called with %s, want %s", tc.legalEntityStore.getByIDArg, tc.wantLegalEntityID)
			}
		})
	}
}

func TestCustomerProfileService_Get(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		id             string
		storeProfile   *core.CustomerProfile
		storeErr       error
		wantErr        string
		wantProfile    bool
		expectedID     string
		expectedStatus string
	}{
		{
			name: "returns customer profile when found",
			id:   "cus_123",
			storeProfile: &core.CustomerProfile{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "USD",
				Notes:           "Preferred customer",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantProfile:    true,
			expectedID:     "cus_123",
			expectedStatus: string(core.CustomerProfileStatusActive),
		},
		{
			name:     "returns not found error",
			id:       "cus_nonexistent",
			storeErr: ErrCustomerProfileNotFound,
			wantErr:  "not found",
		},
		{
			name:     "propagates store errors",
			id:       "cus_123",
			storeErr: errors.New("database error"),
			wantErr:  "database error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &customerProfileStoreStub{
				getByIDRes: tc.storeProfile,
				getByIDErr: tc.storeErr,
			}
			svc := NewCustomerProfileService(nil, store)

			got, err := svc.Get(context.Background(), tc.id)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Get() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Get() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}

			if tc.wantProfile {
				if store.getByIDArg != tc.id {
					t.Errorf("GetByID called with %s, want %s", store.getByIDArg, tc.id)
				}
				if got.ID != tc.expectedID {
					t.Errorf("got ID = %s, want %s", got.ID, tc.expectedID)
				}
				if got.Status != tc.expectedStatus {
					t.Errorf("got Status = %s, want %s", got.Status, tc.expectedStatus)
				}
			}
		})
	}
}

func TestCustomerProfileService_Update(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	currency := "EUR"
	notes := ""
	status := "inactive"

	tests := []struct {
		name          string
		id            string
		cmd           PatchCustomerProfileCommand
		storeProfile  *core.CustomerProfile
		storeErr      error
		wantErr       string
		wantUpdated   bool
		updatedFields core.CustomerProfile
	}{
		{
			name: "applies partial patch successfully",
			id:   "cus_123",
			cmd: PatchCustomerProfileCommand{
				DefaultCurrency: &currency,
				Notes:           &notes,
			},
			storeProfile: &core.CustomerProfile{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "USD",
				Notes:           "Old notes",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantUpdated: true,
			updatedFields: core.CustomerProfile{
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "EUR",
				Notes:           "",
			},
		},
		{
			name: "updates status",
			id:   "cus_123",
			cmd: PatchCustomerProfileCommand{
				Status: &status,
			},
			storeProfile: &core.CustomerProfile{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantUpdated: true,
			updatedFields: core.CustomerProfile{
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusInactive,
				DefaultCurrency: "USD",
			},
		},
		{
			name:     "propagates not-found error",
			id:       "cus_nonexistent",
			cmd:      PatchCustomerProfileCommand{DefaultCurrency: &currency},
			storeErr: ErrCustomerProfileNotFound,
			wantErr:  "not found",
		},
		{
			name: "rejects patch that would make default currency blank",
			id:   "cus_123",
			cmd:  PatchCustomerProfileCommand{DefaultCurrency: ptr("")},
			storeProfile: &core.CustomerProfile{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantErr: "default currency",
		},
		{
			name: "rejects patch that would set invalid status",
			id:   "cus_123",
			cmd:  PatchCustomerProfileCommand{Status: ptr("invalid_status")},
			storeProfile: &core.CustomerProfile{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantErr: "invalid customer profile status",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &customerProfileStoreStub{
				getByIDRes: tc.storeProfile,
				getByIDErr: tc.storeErr,
			}
			svc := NewCustomerProfileService(nil, store)

			got, err := svc.Update(context.Background(), tc.id, tc.cmd)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Update() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Update() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if tc.wantUpdated {
				if store.saveArg == nil {
					t.Fatal("store.Save not called")
				}
				if store.getByIDArg != tc.id {
					t.Errorf("GetByID called with %s, want %s", store.getByIDArg, tc.id)
				}
				if store.saveArg.DefaultCurrency != tc.updatedFields.DefaultCurrency {
					t.Errorf("saved DefaultCurrency = %v, want %v", store.saveArg.DefaultCurrency, tc.updatedFields.DefaultCurrency)
				}
				if store.saveArg.Notes != tc.updatedFields.Notes {
					t.Errorf("saved Notes = %v, want %v", store.saveArg.Notes, tc.updatedFields.Notes)
				}
				if store.saveArg.Status != tc.updatedFields.Status {
					t.Errorf("saved Status = %v, want %v", store.saveArg.Status, tc.updatedFields.Status)
				}
				if got.ID == "" {
					t.Error("returned customer profile ID is empty")
				}
			}
		})
	}
}

func TestCustomerProfileService_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		id           string
		storeProfile *core.CustomerProfile
		storeErr     error
		wantErr      string
		wantDeleted  bool
	}{
		{
			name: "deletes existing customer profile",
			id:   "cus_123",
			storeProfile: &core.CustomerProfile{
				ID:              "cus_123",
				LegalEntityID:   "le_456",
				Status:          core.CustomerProfileStatusActive,
				DefaultCurrency: "USD",
			},
			wantDeleted: true,
		},
		{
			name:     "propagates not-found error",
			id:       "cus_nonexistent",
			storeErr: ErrCustomerProfileNotFound,
			wantErr:  "not found",
		},
		{
			name:     "propagates store errors",
			id:       "cus_123",
			storeErr: errors.New("database error"),
			wantErr:  "database error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &customerProfileStoreStub{
				getByIDRes: tc.storeProfile,
				getByIDErr: tc.storeErr,
			}
			svc := NewCustomerProfileService(nil, store)

			err := svc.Delete(context.Background(), tc.id)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Delete() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Delete() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Delete() error = %v", err)
			}

			if tc.wantDeleted {
				if store.getByIDArg != tc.id {
					t.Errorf("GetByID called with %s, want %s", store.getByIDArg, tc.id)
				}
				if store.deleteArg != tc.id {
					t.Errorf("Delete called with %s, want %s", store.deleteArg, tc.id)
				}
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
