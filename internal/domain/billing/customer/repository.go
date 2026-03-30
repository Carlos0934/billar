package customer

import (
	"context"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

type Repository interface {
	Save(ctx context.Context, customer *Customer) error
	GetByID(ctx context.Context, id billingvalues.CustomerID) (*Customer, error)
}
