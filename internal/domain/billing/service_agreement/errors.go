package serviceagreement

import "errors"

var (
	ErrServiceAgreementIDRequired = errors.New("serviceagreement: id is required")
	ErrCustomerIDRequired         = errors.New("serviceagreement: customer id is required")
	ErrBillingModeMustBeHourly    = errors.New("serviceagreement: billing mode must be hourly")
	ErrHourlyRateMustBePositive   = errors.New("serviceagreement: hourly rate must be positive")
	ErrCreatedAtRequired          = errors.New("serviceagreement: created at is required")
	ErrActivationTimeRequired     = errors.New("serviceagreement: activation time is required")
	ErrDeactivationTimeRequired   = errors.New("serviceagreement: deactivation time is required")
	ErrRateChangeTimeRequired     = errors.New("serviceagreement: rate change time is required")
)
