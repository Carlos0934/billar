package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

type issuerProfileStoreStub struct {
	saveArg    *core.IssuerProfile
	saveErr    error
	getByIDArg string
	getByIDRes *core.IssuerProfile
	getByIDErr error
}

func (s *issuerProfileStoreStub) Save(ctx context.Context, profile *core.IssuerProfile) error {
	_ = ctx
	s.saveArg = profile
	return s.saveErr
}

func (s *issuerProfileStoreStub) GetByID(ctx context.Context, id string) (*core.IssuerProfile, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

type legalEntityStoreStubForIssuer struct {
	called     bool
	query      ListQuery
	result     ListResult[core.LegalEntity]
	err        error
	saveArg    *core.LegalEntity
	saveErr    error
	getByIDArg string
	getByIDRes *core.LegalEntity
	getByIDErr error
	deleteArg  string
	deleteErr  error
}

func (s *legalEntityStoreStubForIssuer) List(ctx context.Context, query ListQuery) (ListResult[core.LegalEntity], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func (s *legalEntityStoreStubForIssuer) Save(ctx context.Context, entity *core.LegalEntity) error {
	_ = ctx
	s.saveArg = entity
	return s.saveErr
}

func (s *legalEntityStoreStubForIssuer) GetByID(ctx context.Context, id string) (*core.LegalEntity, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *legalEntityStoreStubForIssuer) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteArg = id
	return s.deleteErr
}

func TestIssuerProfileService_Create(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		cmd               CreateIssuerProfileCommand
		legalEntityStore  *legalEntityStoreStubForIssuer
		issuerStore       *issuerProfileStoreStub
		wantErr           string
		wantSaved         bool
		savedFields       core.IssuerProfile
		wantLegalEntityID string
	}{
		{
			name: "creates issuer profile with valid legal entity",
			cmd: CreateIssuerProfileCommand{
				LegalEntityID:   "le_123",
				DefaultCurrency: "USD",
				DefaultNotes:    "Payment terms: Net 30",
			},
			legalEntityStore: &legalEntityStoreStubForIssuer{
				getByIDRes: &core.LegalEntity{
					ID:        "le_123",
					Type:      core.EntityTypeCompany,
					LegalName: "Acme SRL",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			issuerStore: &issuerProfileStoreStub{},
			wantSaved:   true,
			savedFields: core.IssuerProfile{
				LegalEntityID:   "le_123",
				DefaultCurrency: "USD",
				DefaultNotes:    "Payment terms: Net 30",
			},
			wantLegalEntityID: "le_123",
		},
		{
			name: "rejects non-existent legal entity",
			cmd: CreateIssuerProfileCommand{
				LegalEntityID:   "le_nonexistent",
				DefaultCurrency: "USD",
			},
			legalEntityStore: &legalEntityStoreStubForIssuer{
				getByIDErr: ErrLegalEntityNotFound,
			},
			issuerStore: &issuerProfileStoreStub{},
			wantErr:     "legal entity not found",
		},
		{
			name: "rejects missing legal entity id",
			cmd: CreateIssuerProfileCommand{
				DefaultCurrency: "USD",
			},
			legalEntityStore: &legalEntityStoreStubForIssuer{},
			issuerStore:      &issuerProfileStoreStub{},
			wantErr:          "legal entity id is required",
		},
		{
			name: "rejects missing default currency",
			cmd: CreateIssuerProfileCommand{
				LegalEntityID: "le_123",
			},
			legalEntityStore: &legalEntityStoreStubForIssuer{},
			issuerStore:      &issuerProfileStoreStub{},
			wantErr:          "default currency is required",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewIssuerProfileService(tc.legalEntityStore, tc.issuerStore)

			got, err := svc.Create(context.Background(), tc.cmd)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Create() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Create() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				if tc.issuerStore.saveArg != nil {
					t.Fatalf("issuer store.Save called unexpectedly, arg = %+v", tc.issuerStore.saveArg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if tc.wantSaved && tc.issuerStore.saveArg == nil {
				t.Fatal("issuer store.Save not called")
			}

			if tc.wantSaved {
				if tc.issuerStore.saveArg.LegalEntityID != tc.savedFields.LegalEntityID {
					t.Errorf("saved LegalEntityID = %v, want %v", tc.issuerStore.saveArg.LegalEntityID, tc.savedFields.LegalEntityID)
				}
				if tc.issuerStore.saveArg.DefaultCurrency != tc.savedFields.DefaultCurrency {
					t.Errorf("saved DefaultCurrency = %v, want %v", tc.issuerStore.saveArg.DefaultCurrency, tc.savedFields.DefaultCurrency)
				}
				if tc.issuerStore.saveArg.ID == "" {
					t.Error("saved issuer profile ID is empty")
				}
				if got.ID == "" {
					t.Error("returned issuer profile ID is empty")
				}
			}

			if tc.wantLegalEntityID != "" && tc.legalEntityStore.getByIDArg != tc.wantLegalEntityID {
				t.Errorf("legal entity store.GetByID called with %s, want %s", tc.legalEntityStore.getByIDArg, tc.wantLegalEntityID)
			}
		})
	}
}

func TestIssuerProfileService_Get(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		id               string
		storeProfile     *core.IssuerProfile
		storeErr         error
		wantErr          string
		wantProfile      bool
		expectedID       string
		expectedCurrency string
	}{
		{
			name: "returns issuer profile when found",
			id:   "iss_123",
			storeProfile: &core.IssuerProfile{
				ID:              "iss_123",
				LegalEntityID:   "le_456",
				DefaultCurrency: "USD",
				DefaultNotes:    "Payment terms: Net 30",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantProfile:      true,
			expectedID:       "iss_123",
			expectedCurrency: "USD",
		},
		{
			name:     "returns not found error",
			id:       "iss_nonexistent",
			storeErr: ErrIssuerProfileNotFound,
			wantErr:  "not found",
		},
		{
			name:     "propagates store errors",
			id:       "iss_123",
			storeErr: errors.New("database error"),
			wantErr:  "database error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &issuerProfileStoreStub{
				getByIDRes: tc.storeProfile,
				getByIDErr: tc.storeErr,
			}
			svc := NewIssuerProfileService(nil, store)

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
				if got.DefaultCurrency != tc.expectedCurrency {
					t.Errorf("got DefaultCurrency = %s, want %s", got.DefaultCurrency, tc.expectedCurrency)
				}
			}
		})
	}
}

func TestIssuerProfileService_Update(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	currency := "EUR"
	notes := "Updated payment terms"

	tests := []struct {
		name          string
		id            string
		cmd           PatchIssuerProfileCommand
		storeProfile  *core.IssuerProfile
		storeErr      error
		wantErr       string
		wantUpdated   bool
		updatedFields core.IssuerProfile
	}{
		{
			name: "applies partial patch successfully",
			id:   "iss_123",
			cmd: PatchIssuerProfileCommand{
				DefaultCurrency: &currency,
				DefaultNotes:    &notes,
			},
			storeProfile: &core.IssuerProfile{
				ID:              "iss_123",
				LegalEntityID:   "le_456",
				DefaultCurrency: "USD",
				DefaultNotes:    "Old notes",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantUpdated: true,
			updatedFields: core.IssuerProfile{
				LegalEntityID:   "le_456",
				DefaultCurrency: "EUR",
				DefaultNotes:    "Updated payment terms",
			},
		},
		{
			name:     "propagates not-found error",
			id:       "iss_nonexistent",
			cmd:      PatchIssuerProfileCommand{DefaultCurrency: &currency},
			storeErr: ErrIssuerProfileNotFound,
			wantErr:  "not found",
		},
		{
			name: "rejects patch that would make default currency blank",
			id:   "iss_123",
			cmd:  PatchIssuerProfileCommand{DefaultCurrency: ptr("")},
			storeProfile: &core.IssuerProfile{
				ID:              "iss_123",
				LegalEntityID:   "le_456",
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantErr: "default currency",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &issuerProfileStoreStub{
				getByIDRes: tc.storeProfile,
				getByIDErr: tc.storeErr,
			}
			svc := NewIssuerProfileService(nil, store)

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
				if store.saveArg.DefaultNotes != tc.updatedFields.DefaultNotes {
					t.Errorf("saved DefaultNotes = %v, want %v", store.saveArg.DefaultNotes, tc.updatedFields.DefaultNotes)
				}
				if got.ID == "" {
					t.Error("returned issuer profile ID is empty")
				}
			}
		})
	}
}
