package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrCustomerListAccessDenied = errors.New("customer list requires an authenticated identity")

type CustomerStore interface {
	List(ctx context.Context, query ListQuery) (ListResult[core.Customer], error)
}

type CustomerService struct {
	identities AuthenticatedIdentitySource
	store      CustomerStore
}

func NewCustomerService(identities AuthenticatedIdentitySource, store CustomerStore) CustomerService {
	return CustomerService{identities: identities, store: store}
}

func (s CustomerService) List(ctx context.Context, query ListQuery) (ListResult[CustomerDTO], error) {
	query = query.Normalize()

	if s.identities == nil {
		return ListResult[CustomerDTO]{}, errors.New("customer authenticated identity source is required")
	}
	if s.store == nil {
		return ListResult[CustomerDTO]{}, errors.New("customer store is required")
	}

	_, ok, err := s.identities.CurrentIdentity(ctx)
	if err != nil {
		return ListResult[CustomerDTO]{}, fmt.Errorf("load authenticated identity: %w", err)
	}
	if !ok {
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
