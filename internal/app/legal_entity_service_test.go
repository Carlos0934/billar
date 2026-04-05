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

type legalEntityStoreStub struct {
	called bool
	query  ListQuery
	result ListResult[core.LegalEntity]
	err    error
	// Write operations
	saveArg    *core.LegalEntity
	saveErr    error
	getByIDArg string
	getByIDRes *core.LegalEntity
	getByIDErr error
	deleteArg  string
	deleteErr  error
}

func (s *legalEntityStoreStub) List(ctx context.Context, query ListQuery) (ListResult[core.LegalEntity], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func (s *legalEntityStoreStub) Save(ctx context.Context, entity *core.LegalEntity) error {
	_ = ctx
	s.saveArg = entity
	return s.saveErr
}

func (s *legalEntityStoreStub) GetByID(ctx context.Context, id string) (*core.LegalEntity, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *legalEntityStoreStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteArg = id
	return s.deleteErr
}

func TestLegalEntityService_List(t *testing.T) {
	t.Parallel()

	createdAt := "2026-04-03T10:00:00Z"
	updatedAt := "2026-04-03T10:05:00Z"

	tests := []struct {
		name         string
		query        ListQuery
		storeResult  ListResult[core.LegalEntity]
		storeErr     error
		wantQuery    ListQuery
		wantResult   ListResult[LegalEntityDTO]
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
			storeResult: ListResult[core.LegalEntity]{
				Items: []core.LegalEntity{{
					ID:             "le_123",
					Type:           core.EntityTypeCompany,
					LegalName:      "Acme SRL",
					TradeName:      "Acme",
					TaxID:          "001-1234567-8",
					Email:          "billing@acme.example",
					Phone:          "+1 809 555 0101",
					Website:        "https://acme.example",
					BillingAddress: core.Address{Street: "Calle 1", City: "Santo Domingo"},
					CreatedAt:      time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
					UpdatedAt:      time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
				}},
				Total:    1,
				Page:     1,
				PageSize: 100,
			},
			wantQuery: ListQuery{Page: 1, PageSize: 100, Search: "Acme", SortField: "legal_name", SortDir: "desc"},
			wantResult: ListResult[LegalEntityDTO]{
				Items: []LegalEntityDTO{{
					ID:             "le_123",
					Type:           string(core.EntityTypeCompany),
					LegalName:      "Acme SRL",
					TradeName:      "Acme",
					TaxID:          "001-1234567-8",
					Email:          "billing@acme.example",
					Phone:          "+1 809 555 0101",
					Website:        "https://acme.example",
					BillingAddress: AddressDTO{Street: "Calle 1", City: "Santo Domingo"},
					CreatedAt:      createdAt,
					UpdatedAt:      updatedAt,
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
			storeResult:  ListResult[core.LegalEntity]{},
			storeErr:     errors.New("database error"),
			wantErr:      "database error",
			wantStoreHit: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &legalEntityStoreStub{result: tc.storeResult, err: tc.storeErr}
			svc := NewLegalEntityService(store)

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

func TestLegalEntityService_Create(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cmd         CreateLegalEntityCommand
		wantErr     string
		wantSaved   bool
		savedFields core.LegalEntity
	}{
		{
			name: "creates company legal entity with valid command",
			cmd: CreateLegalEntityCommand{
				Type:           "company",
				LegalName:      "Acme SRL",
				TradeName:      "Acme",
				TaxID:          "001-1234567-8",
				Email:          "billing@acme.example",
				Phone:          "+1 809 555 0101",
				Website:        "https://acme.example",
				BillingAddress: AddressDTO{Street: "Calle 1", City: "Santo Domingo"},
			},
			wantSaved: true,
			savedFields: core.LegalEntity{
				Type:           core.EntityTypeCompany,
				LegalName:      "Acme SRL",
				TradeName:      "Acme",
				TaxID:          "001-1234567-8",
				Email:          "billing@acme.example",
				Phone:          "+1 809 555 0101",
				Website:        "https://acme.example",
				BillingAddress: core.Address{Street: "Calle 1", City: "Santo Domingo"},
			},
		},
		{
			name: "creates individual legal entity with valid command",
			cmd: CreateLegalEntityCommand{
				Type:      "individual",
				LegalName: "John Doe",
				Email:     "john@example.com",
			},
			wantSaved: true,
			savedFields: core.LegalEntity{
				Type:      core.EntityTypeIndividual,
				LegalName: "John Doe",
				Email:     "john@example.com",
			},
		},
		{
			name:    "rejects missing legal_name",
			cmd:     CreateLegalEntityCommand{Type: "company"},
			wantErr: "legal name is required",
		},
		{
			name:    "rejects invalid entity type",
			cmd:     CreateLegalEntityCommand{Type: "invalid_type", LegalName: "Test Company"},
			wantErr: "invalid entity type",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &legalEntityStoreStub{}
			svc := NewLegalEntityService(store)

			got, err := svc.Create(context.Background(), tc.cmd)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("Create() error = nil, want non-nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Create() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				if store.saveArg != nil {
					t.Fatalf("store.Save called unexpectedly, arg = %+v", store.saveArg)
				}
				return
			}

			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if tc.wantSaved && store.saveArg == nil {
				t.Fatal("store.Save not called")
			}

			if tc.wantSaved {
				if store.saveArg.Type != tc.savedFields.Type {
					t.Errorf("saved Type = %v, want %v", store.saveArg.Type, tc.savedFields.Type)
				}
				if store.saveArg.LegalName != tc.savedFields.LegalName {
					t.Errorf("saved LegalName = %v, want %v", store.saveArg.LegalName, tc.savedFields.LegalName)
				}
				if store.saveArg.ID == "" {
					t.Error("saved legal entity ID is empty")
				}
				if got.ID == "" {
					t.Error("returned legal entity ID is empty")
				}
			}
		})
	}
}

func TestLegalEntityService_Get(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		id           string
		storeEntity  *core.LegalEntity
		storeErr     error
		wantErr      string
		wantEntity   bool
		expectedID   string
		expectedName string
	}{
		{
			name: "returns legal entity when found",
			id:   "le_123",
			storeEntity: &core.LegalEntity{
				ID:             "le_123",
				Type:           core.EntityTypeCompany,
				LegalName:      "Acme SRL",
				TradeName:      "Acme",
				TaxID:          "001-1234567-8",
				Email:          "billing@acme.example",
				Phone:          "+1 809 555 0101",
				Website:        "https://acme.example",
				BillingAddress: core.Address{Street: "Calle 1", City: "Santo Domingo"},
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			wantEntity:   true,
			expectedID:   "le_123",
			expectedName: "Acme SRL",
		},
		{
			name:     "returns not found error",
			id:       "le_nonexistent",
			storeErr: ErrLegalEntityNotFound,
			wantErr:  "not found",
		},
		{
			name:     "propagates store errors",
			id:       "le_123",
			storeErr: errors.New("database error"),
			wantErr:  "database error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &legalEntityStoreStub{
				getByIDRes: tc.storeEntity,
				getByIDErr: tc.storeErr,
			}
			svc := NewLegalEntityService(store)

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

			if tc.wantEntity {
				if store.getByIDArg != tc.id {
					t.Errorf("GetByID called with %s, want %s", store.getByIDArg, tc.id)
				}
				if got.ID != tc.expectedID {
					t.Errorf("got ID = %s, want %s", got.ID, tc.expectedID)
				}
				if got.LegalName != tc.expectedName {
					t.Errorf("got LegalName = %s, want %s", got.LegalName, tc.expectedName)
				}
			}
		})
	}
}

func TestLegalEntityService_Update(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	email := "updated@example.com"
	tradeName := ""

	tests := []struct {
		name          string
		id            string
		cmd           PatchLegalEntityCommand
		storeEntity   *core.LegalEntity
		storeErr      error
		wantErr       string
		wantUpdated   bool
		updatedFields core.LegalEntity
	}{
		{
			name: "applies partial patch successfully",
			id:   "le_123",
			cmd: PatchLegalEntityCommand{
				Email:     &email,
				TradeName: &tradeName,
			},
			storeEntity: &core.LegalEntity{
				ID:        "le_123",
				Type:      core.EntityTypeCompany,
				LegalName: "Acme SRL",
				TradeName: "Old Trade",
				Email:     "old@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantUpdated: true,
			updatedFields: core.LegalEntity{
				Type:      core.EntityTypeCompany,
				LegalName: "Acme SRL",
				Email:     "updated@example.com",
				TradeName: "",
			},
		},
		{
			name:     "propagates not-found error",
			id:       "le_nonexistent",
			cmd:      PatchLegalEntityCommand{Email: &email},
			storeErr: ErrLegalEntityNotFound,
			wantErr:  "not found",
		},
		{
			name: "rejects patch that would make legal name blank",
			id:   "le_123",
			cmd:  PatchLegalEntityCommand{LegalName: ptr("")},
			storeEntity: &core.LegalEntity{
				ID:        "le_123",
				Type:      core.EntityTypeCompany,
				LegalName: "Acme SRL",
				Email:     "old@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: "legal name",
		},
		{
			name: "rejects patch that would set invalid type",
			id:   "le_123",
			cmd:  PatchLegalEntityCommand{Type: ptr("invalid_type")},
			storeEntity: &core.LegalEntity{
				ID:        "le_123",
				Type:      core.EntityTypeCompany,
				LegalName: "Acme SRL",
				Email:     "old@example.com",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantErr: "invalid entity type",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &legalEntityStoreStub{
				getByIDRes: tc.storeEntity,
				getByIDErr: tc.storeErr,
			}
			svc := NewLegalEntityService(store)

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
				if store.saveArg.Email != tc.updatedFields.Email {
					t.Errorf("saved Email = %v, want %v", store.saveArg.Email, tc.updatedFields.Email)
				}
				if store.saveArg.TradeName != tc.updatedFields.TradeName {
					t.Errorf("saved TradeName = %v, want %v", store.saveArg.TradeName, tc.updatedFields.TradeName)
				}
				if got.ID == "" {
					t.Error("returned legal entity ID is empty")
				}
			}
		})
	}
}

func TestLegalEntityService_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		id          string
		storeEntity *core.LegalEntity
		storeErr    error
		wantErr     string
		wantDeleted bool
	}{
		{
			name: "deletes existing legal entity",
			id:   "le_123",
			storeEntity: &core.LegalEntity{
				ID:        "le_123",
				Type:      core.EntityTypeCompany,
				LegalName: "Acme SRL",
			},
			wantDeleted: true,
		},
		{
			name:     "propagates not-found error",
			id:       "le_nonexistent",
			storeErr: ErrLegalEntityNotFound,
			wantErr:  "not found",
		},
		{
			name:     "propagates store errors",
			id:       "le_123",
			storeErr: errors.New("database error"),
			wantErr:  "database error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &legalEntityStoreStub{
				getByIDRes: tc.storeEntity,
				getByIDErr: tc.storeErr,
			}
			svc := NewLegalEntityService(store)

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
