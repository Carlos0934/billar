package serviceagreement

import (
	"context"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

type Repository interface {
	Save(ctx context.Context, agreement *ServiceAgreement) error
	GetByID(ctx context.Context, id billingvalues.ServiceAgreementID) (*ServiceAgreement, error)
	ListByCustomerID(ctx context.Context, customerID billingvalues.CustomerID) ([]*ServiceAgreement, error)
}
