package app

import (
	"context"

	"github.com/Carlos0934/billar/internal/core"
)

// noopServiceAgreementStore is a ServiceAgreementStore implementation that
// always returns ErrServiceAgreementNotFound for reads and discards writes.
// It is used as a placeholder in entrypoints when the real SQLite store is
// not yet wired (deferred infrastructure phase).
type noopServiceAgreementStore struct{}

func (noopServiceAgreementStore) Save(_ context.Context, _ *core.ServiceAgreement) error {
	return ErrServiceAgreementNotFound
}

func (noopServiceAgreementStore) GetByID(_ context.Context, _ string) (*core.ServiceAgreement, error) {
	return nil, ErrServiceAgreementNotFound
}

func (noopServiceAgreementStore) ListByCustomerProfileID(_ context.Context, _ string) ([]core.ServiceAgreement, error) {
	return nil, nil
}

// noopCustomerProfileStoreForAgreement is a minimal CustomerProfileStore stub
// used by NewNoopAgreementService to avoid a nil panic in AgreementService.Create.
type noopCustomerProfileStoreForAgreement struct{}

func (noopCustomerProfileStoreForAgreement) Save(_ context.Context, _ *core.CustomerProfile) error {
	return ErrCustomerProfileNotFound
}

func (noopCustomerProfileStoreForAgreement) GetByID(_ context.Context, _ string) (*core.CustomerProfile, error) {
	return nil, ErrCustomerProfileNotFound
}

func (noopCustomerProfileStoreForAgreement) List(_ context.Context, _ ListQuery) (ListResult[core.CustomerProfile], error) {
	return ListResult[core.CustomerProfile]{}, nil
}

func (noopCustomerProfileStoreForAgreement) Delete(_ context.Context, _ string) error {
	return ErrCustomerProfileNotFound
}

// NewNoopAgreementService returns an AgreementService backed by noop stores.
// All read operations return ErrServiceAgreementNotFound; writes fail gracefully.
// This is used in entrypoints before the SQLite agreement store is wired.
func NewNoopAgreementService() AgreementService {
	return NewAgreementService(noopServiceAgreementStore{}, noopCustomerProfileStoreForAgreement{})
}
