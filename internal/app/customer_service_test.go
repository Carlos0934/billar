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

type customerIdentitySourceStub struct {
	identity AuthenticatedIdentity
	ok       bool
	err      error
}

func (s customerIdentitySourceStub) CurrentIdentity(context.Context) (AuthenticatedIdentity, bool, error) {
	return s.identity, s.ok, s.err
}

type customerStoreStub struct {
	called bool
	query  ListQuery
	result ListResult[core.Customer]
	err    error
	// Write operations
	saveArg    *core.Customer
	saveErr    error
	getByIDArg string
	getByIDRes *core.Customer
	getByIDErr error
	deleteArg  string
	deleteErr  error
}

func (s *customerStoreStub) List(ctx context.Context, query ListQuery) (ListResult[core.Customer], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func (s *customerStoreStub) Save(ctx context.Context, customer *core.Customer) error {
	_ = ctx
	s.saveArg = customer
	return s.saveErr
}

func (s *customerStoreStub) GetByID(ctx context.Context, id string) (*core.Customer, error) {
	_ = ctx
	s.getByIDArg = id
	return s.getByIDRes, s.getByIDErr
}

func (s *customerStoreStub) Delete(ctx context.Context, id string) error {
	_ = ctx
	s.deleteArg = id
	return s.deleteErr
}

func TestCustomerServiceList(t *testing.T) {
	t.Parallel()

	createdAt := "2026-04-03T10:00:00Z"
	updatedAt := "2026-04-03T10:05:00Z"

	tests := []struct {
		name         string
		identityOK   bool
		query        ListQuery
		storeResult  ListResult[core.Customer]
		wantQuery    ListQuery
		wantResult   ListResult[CustomerDTO]
		wantErr      string
		wantStoreHit bool
	}{
		{
			name:       "returns mapped list for authenticated identity",
			identityOK: true,
			query: ListQuery{
				Page:      0,
				PageSize:  500,
				Search:    "  Acme  ",
				SortField: " name ",
				SortDir:   " DESC ",
			},
			storeResult: ListResult[core.Customer]{
				Items: []core.Customer{{
					ID:              "cus_123",
					Type:            core.CustomerTypeCompany,
					LegalName:       "Acme SRL",
					TradeName:       "Acme",
					TaxID:           "001-1234567-8",
					Email:           "billing@acme.example",
					Phone:           "+1 809 555 0101",
					Website:         "https://acme.example",
					BillingAddress:  core.Address{Street: "Calle 1", City: "Santo Domingo"},
					Status:          core.CustomerStatusActive,
					DefaultCurrency: "USD",
					Notes:           "Preferred by email",
					CreatedAt:       time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
					UpdatedAt:       time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC),
				}},
				Total:    1,
				Page:     1,
				PageSize: 100,
			},
			wantQuery: ListQuery{Page: 1, PageSize: 100, Search: "Acme", SortField: "legal_name", SortDir: "desc"},
			wantResult: ListResult[CustomerDTO]{
				Items: []CustomerDTO{{
					ID:              "cus_123",
					Type:            string(core.CustomerTypeCompany),
					LegalName:       "Acme SRL",
					TradeName:       "Acme",
					TaxID:           "001-1234567-8",
					Email:           "billing@acme.example",
					Phone:           "+1 809 555 0101",
					Website:         "https://acme.example",
					BillingAddress:  AddressDTO{Street: "Calle 1", City: "Santo Domingo"},
					Status:          string(core.CustomerStatusActive),
					DefaultCurrency: "USD",
					Notes:           "Preferred by email",
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
			name:         "rejects missing identity before hitting store",
			query:        ListQuery{Page: 3, PageSize: 5, Search: "Acme"},
			wantErr:      "authenticated identity",
			wantResult:   ListResult[CustomerDTO]{},
			wantQuery:    ListQuery{},
			wantStoreHit: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identities := customerIdentitySourceStub{identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true}, ok: tc.identityOK}
			store := &customerStoreStub{result: tc.storeResult}
			svc := NewCustomerService(identities, store)

			got, err := svc.List(context.Background(), tc.query)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("List() error = nil, want non-nil")
				}
				if !errors.Is(err, ErrCustomerListAccessDenied) && !strings.Contains(err.Error(), tc.wantErr) {
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

func TestCustomerServiceCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		identityOK  bool
		cmd         CreateCustomerCommand
		wantErr     string
		wantSaved   bool
		savedFields core.Customer
	}{
		{
			name:       "creates customer with valid command",
			identityOK: true,
			cmd: CreateCustomerCommand{
				Type:      "company",
				LegalName: "Acme SRL",
				TradeName: "Acme",
				TaxID:     "001-1234567-8",
				Email:     "billing@acme.example",
				Phone:     "+1 809 555 0101",
				Website:   "https://acme.example",
				Notes:     "Preferred customer",
			},
			wantSaved: true,
			savedFields: core.Customer{
				Type:      core.CustomerTypeCompany,
				LegalName: "Acme SRL",
				TradeName: "Acme",
				TaxID:     "001-1234567-8",
				Email:     "billing@acme.example",
				Phone:     "+1 809 555 0101",
				Website:   "https://acme.example",
				Notes:     "Preferred customer",
			},
		},
		{
			name:       "rejects missing legal_name",
			identityOK: true,
			cmd: CreateCustomerCommand{
				Type: "company",
			},
			wantErr:   "legal name is required",
			wantSaved: false,
		},
		{
			name:       "rejects invalid customer type",
			identityOK: true,
			cmd: CreateCustomerCommand{
				Type:      "invalid_type",
				LegalName: "Test Company",
			},
			wantErr:   "invalid customer type",
			wantSaved: false,
		},
		{
			name:       "rejects unauthenticated request",
			identityOK: false,
			cmd: CreateCustomerCommand{
				Type:      "company",
				LegalName: "Test Company",
			},
			wantErr:   "authenticated",
			wantSaved: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identities := customerIdentitySourceStub{
				identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true},
				ok:       tc.identityOK,
			}
			store := &customerStoreStub{}
			svc := NewCustomerService(identities, store)

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
					t.Error("saved customer ID is empty")
				}
				if got.ID == "" {
					t.Error("returned customer ID is empty")
				}
			}
		})
	}
}

func TestCustomerServiceUpdate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC)
	email := "updated@example.com"
	notes := ""

	tests := []struct {
		name          string
		identityOK    bool
		id            string
		cmd           PatchCustomerCommand
		storeCustomer *core.Customer
		storeErr      error
		wantErr       string
		wantUpdated   bool
		updatedFields core.Customer
	}{
		{
			name:       "applies partial patch successfully",
			identityOK: true,
			id:         "cus_123",
			cmd: PatchCustomerCommand{
				Email: &email,
				Notes: &notes,
			},
			storeCustomer: &core.Customer{
				ID:              "cus_123",
				Type:            core.CustomerTypeCompany,
				LegalName:       "Acme SRL",
				Email:           "old@example.com",
				Notes:           "Old notes",
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantUpdated: true,
			updatedFields: core.Customer{
				Type:            core.CustomerTypeCompany,
				LegalName:       "Acme SRL",
				Email:           "updated@example.com",
				Notes:           "",
				DefaultCurrency: "USD",
			},
		},
		{
			name:       "propagates not-found error",
			identityOK: true,
			id:         "cus_nonexistent",
			cmd: PatchCustomerCommand{
				Email: &email,
			},
			storeErr: ErrCustomerNotFound,
			wantErr:  "not found",
		},
		{
			name:       "rejects unauthenticated request",
			identityOK: false,
			id:         "cus_123",
			cmd: PatchCustomerCommand{
				Email: &email,
			},
			wantErr: "authenticated",
		},
		{
			name:       "rejects patch that would make legal name blank",
			identityOK: true,
			id:         "cus_123",
			cmd: PatchCustomerCommand{
				LegalName: ptr(""),
			},
			storeCustomer: &core.Customer{
				ID:              "cus_123",
				Type:            core.CustomerTypeCompany,
				LegalName:       "Acme SRL",
				Email:           "old@example.com",
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantErr: "legal name",
		},
		{
			name:       "rejects patch that would set invalid type",
			identityOK: true,
			id:         "cus_123",
			cmd: PatchCustomerCommand{
				Type: ptr("invalid_type"),
			},
			storeCustomer: &core.Customer{
				ID:              "cus_123",
				Type:            core.CustomerTypeCompany,
				LegalName:       "Acme SRL",
				Email:           "old@example.com",
				DefaultCurrency: "USD",
				CreatedAt:       now,
				UpdatedAt:       now,
			},
			wantErr: "invalid customer type",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identities := customerIdentitySourceStub{
				identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true},
				ok:       tc.identityOK,
			}
			store := &customerStoreStub{
				getByIDRes: tc.storeCustomer,
				getByIDErr: tc.storeErr,
			}
			svc := NewCustomerService(identities, store)

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
				if store.saveArg.Notes != tc.updatedFields.Notes {
					t.Errorf("saved Notes = %v, want %v", store.saveArg.Notes, tc.updatedFields.Notes)
				}
				if got.ID == "" {
					t.Error("returned customer ID is empty")
				}
			}
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestCustomerServiceDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		identityOK    bool
		id            string
		storeCustomer *core.Customer
		storeErr      error
		wantErr       string
		wantDeleted   bool
	}{
		{
			name:       "deletes existing customer",
			identityOK: true,
			id:         "cus_123",
			storeCustomer: &core.Customer{
				ID:              "cus_123",
				Type:            core.CustomerTypeCompany,
				LegalName:       "Acme SRL",
				DefaultCurrency: "USD",
			},
			wantDeleted: true,
		},
		{
			name:       "propagates not-found error",
			identityOK: true,
			id:         "cus_nonexistent",
			storeErr:   ErrCustomerNotFound,
			wantErr:    "not found",
		},
		{
			name:       "rejects unauthenticated request",
			identityOK: false,
			id:         "cus_123",
			wantErr:    "authenticated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			identities := customerIdentitySourceStub{
				identity: AuthenticatedIdentity{Email: "user@example.com", EmailVerified: true},
				ok:       tc.identityOK,
			}
			store := &customerStoreStub{
				getByIDRes: tc.storeCustomer,
				getByIDErr: tc.storeErr,
			}
			svc := NewCustomerService(identities, store)

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
