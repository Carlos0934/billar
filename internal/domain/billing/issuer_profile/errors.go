package issuerprofile

import "errors"

var (
	ErrIssuerProfileIDRequired = errors.New("issuerprofile: id is required")
	ErrLegalNameRequired       = errors.New("issuerprofile: legal name is required")
	ErrTaxIDRequired           = errors.New("issuerprofile: tax id is required")
	ErrBillingAddressRequired  = errors.New("issuerprofile: billing address is required")
	ErrCreatedAtRequired       = errors.New("issuerprofile: created at is required")
	ErrUpdatedAtRequired       = errors.New("issuerprofile: updated at is required")
)
