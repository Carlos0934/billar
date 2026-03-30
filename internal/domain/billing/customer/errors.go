package customer

import "errors"

var (
	ErrCustomerIDRequired       = errors.New("customer: id is required")
	ErrBillingNameRequired      = errors.New("customer: billing name is required")
	ErrCreatedAtRequired        = errors.New("customer: created at is required")
	ErrActivationTimeRequired   = errors.New("customer: activation time is required")
	ErrDeactivationTimeRequired = errors.New("customer: deactivation time is required")
	ErrCustomerTypeInvalid      = errors.New("customer: customer type must be individual or company")
)
