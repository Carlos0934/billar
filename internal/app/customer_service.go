package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrCustomerListAccessDenied = errors.New("customer list requires an active session")

type CustomerStore interface {
	List(ctx context.Context, query ListQuery) (ListResult[core.Customer], error)
}

type CustomerService struct {
	sessions SessionStore
	store    CustomerStore
}

func NewCustomerService(sessions SessionStore, store CustomerStore) CustomerService {
	return CustomerService{sessions: sessions, store: store}
}

func (s CustomerService) List(ctx context.Context, query ListQuery) (ListResult[CustomerDTO], error) {
	query = query.Normalize()

	if s.sessions == nil {
		return ListResult[CustomerDTO]{}, errors.New("customer session store is required")
	}
	if s.store == nil {
		return ListResult[CustomerDTO]{}, errors.New("customer store is required")
	}

	session, err := s.sessions.GetCurrent(ctx)
	if err != nil {
		return ListResult[CustomerDTO]{}, fmt.Errorf("load current session: %w", err)
	}
	if session == nil || session.Status != core.SessionStatusActive {
		return ListResult[CustomerDTO]{}, ErrCustomerListAccessDenied
	}

	result, err := s.store.List(ctx, query)
	if err != nil {
		return ListResult[CustomerDTO]{}, fmt.Errorf("list customers: %w", err)
	}

	items := make([]CustomerDTO, 0, len(result.Items))
	for _, customer := range result.Items {
		items = append(items, customerToDTO(customer))
	}

	return ListResult[CustomerDTO]{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}
