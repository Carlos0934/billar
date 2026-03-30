package billingvalues

import "errors"

var (
	ErrCountryCodeLength          = errors.New("billingvalues: country code must be a 2-letter code")
	ErrCountryCodeLetters         = errors.New("billingvalues: country code must contain only letters")
	ErrAddressCountryRequired     = errors.New("billingvalues: address country is required")
	ErrAddressLine1Required       = errors.New("billingvalues: address line1 is required")
	ErrAddressCityRequired        = errors.New("billingvalues: address city is required")
	ErrStartDateRequired          = errors.New("billingvalues: start date is required")
	ErrEndDateRequired            = errors.New("billingvalues: end date is required")
	ErrEndDateBeforeStartDate     = errors.New("billingvalues: end date must not be earlier than start date")
	ErrEmailAddressInvalid        = errors.New("billingvalues: email address is invalid")
	ErrPhoneNumberRequired        = errors.New("billingvalues: phone number is required")
	ErrTaxIdentifierRequired      = errors.New("billingvalues: tax identifier is required")
	ErrCurrencyCodeLength         = errors.New("billingvalues: currency code must be 3 letters")
	ErrCurrencyCodeLetters        = errors.New("billingvalues: currency code must contain only letters")
	ErrCurrencyRequired           = errors.New("billingvalues: currency is required")
	ErrHoursNegative              = errors.New("billingvalues: hours must not be negative")
	ErrHoursMustBePositive        = errors.New("billingvalues: hours must be greater than zero")
	ErrCustomerIDRequired         = errors.New("billingvalues: customer id is required")
	ErrServiceAgreementIDRequired = errors.New("billingvalues: service agreement id is required")
	ErrIssuerProfileIDRequired    = errors.New("billingvalues: issuer profile id is required")
	ErrTimeEntryIDRequired        = errors.New("billingvalues: time entry id is required")
	ErrInvoiceIDRequired          = errors.New("billingvalues: invoice id is required")
)
