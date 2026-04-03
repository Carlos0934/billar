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

type customerSessionStoreStub struct {
	session *core.Session
	getErr  error
	saved   *core.Session
}

func (s *customerSessionStoreStub) Save(ctx context.Context, session *core.Session) error {
	_ = ctx
	if session == nil {
		s.saved = nil
		s.session = nil
		return nil
	}
	copy := *session
	s.saved = &copy
	s.session = &copy
	return nil
}

func (s *customerSessionStoreStub) GetCurrent(ctx context.Context) (*core.Session, error) {
	_ = ctx
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.session == nil {
		return nil, nil
	}
	copy := *s.session
	return &copy, nil
}

type customerStoreStub struct {
	called bool
	query  ListQuery
	result ListResult[core.Customer]
	err    error
}

func (s *customerStoreStub) List(ctx context.Context, query ListQuery) (ListResult[core.Customer], error) {
	_ = ctx
	s.called = true
	s.query = query
	return s.result, s.err
}

func TestCustomerServiceList(t *testing.T) {
	t.Parallel()

	createdAt := "2026-04-03T10:00:00Z"
	updatedAt := "2026-04-03T10:05:00Z"

	tests := []struct {
		name         string
		session      *core.Session
		query        ListQuery
		storeResult  ListResult[core.Customer]
		wantQuery    ListQuery
		wantResult   ListResult[CustomerDTO]
		wantErr      string
		wantStoreHit bool
	}{
		{
			name:    "returns mapped list for active session",
			session: &core.Session{Status: core.SessionStatusActive},
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
			name:         "rejects inactive session before hitting store",
			session:      &core.Session{Status: core.SessionStatusUnauthenticated},
			query:        ListQuery{Page: 3, PageSize: 5, Search: "Acme"},
			wantErr:      "active session",
			wantResult:   ListResult[CustomerDTO]{},
			wantQuery:    ListQuery{},
			wantStoreHit: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sessions := &customerSessionStoreStub{session: tc.session}
			store := &customerStoreStub{result: tc.storeResult}
			svc := NewCustomerService(sessions, store)

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
