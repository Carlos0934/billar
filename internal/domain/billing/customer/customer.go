package customer

import (
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

type CreateParams struct {
	ID              billingvalues.CustomerID
	Type            CustomerType
	BillingName     string
	TradeName       string
	TaxID           *billingvalues.TaxIdentifier
	Email           *billingvalues.EmailAddress
	Phone           *billingvalues.PhoneNumber
	Website         string
	BillingAddress  *billingvalues.Address
	DefaultCurrency billingvalues.CurrencyCode
	Notes           string
	CreatedAt       time.Time
}

type Customer struct {
	id              billingvalues.CustomerID
	customerType    CustomerType
	billingName     string
	tradeName       string
	taxID           *billingvalues.TaxIdentifier
	email           *billingvalues.EmailAddress
	phone           *billingvalues.PhoneNumber
	website         string
	billingAddress  *billingvalues.Address
	status          CustomerStatus
	defaultCurrency billingvalues.CurrencyCode
	notes           string
	createdAt       time.Time
	updatedAt       time.Time
}

func New(params CreateParams) (*Customer, error) {
	if params.ID.IsZero() {
		return nil, ErrCustomerIDRequired
	}
	if err := params.Type.validate(); err != nil {
		return nil, err
	}

	billingName := strings.TrimSpace(params.BillingName)
	if billingName == "" {
		return nil, ErrBillingNameRequired
	}
	if params.CreatedAt.IsZero() {
		return nil, ErrCreatedAtRequired
	}

	defaultCurrency := params.DefaultCurrency
	if defaultCurrency.IsZero() {
		defaultCurrency = billingvalues.DefaultCurrencyCode()
	}

	return &Customer{
		id:              params.ID,
		customerType:    params.Type,
		billingName:     billingName,
		tradeName:       strings.TrimSpace(params.TradeName),
		taxID:           params.TaxID,
		email:           params.Email,
		phone:           params.Phone,
		website:         strings.TrimSpace(params.Website),
		billingAddress:  params.BillingAddress,
		status:          StatusActive,
		defaultCurrency: defaultCurrency,
		notes:           strings.TrimSpace(params.Notes),
		createdAt:       params.CreatedAt.UTC(),
		updatedAt:       params.CreatedAt.UTC(),
	}, nil
}

func (customer *Customer) ID() billingvalues.CustomerID {
	return customer.id
}

func (customer *Customer) BillingName() string {
	return customer.billingName
}

func (customer *Customer) DefaultCurrency() billingvalues.CurrencyCode {
	return customer.defaultCurrency
}

func (customer *Customer) Status() CustomerStatus {
	return customer.status
}

func (customer *Customer) CreatedAt() time.Time {
	return customer.createdAt
}

func (customer *Customer) UpdatedAt() time.Time {
	return customer.updatedAt
}

func (customer *Customer) IsInvoiceReady() bool {
	return customer.status == StatusActive && customer.billingName != ""
}

func (customer *Customer) Activate(at time.Time) error {
	if at.IsZero() {
		return ErrActivationTimeRequired
	}

	customer.status = StatusActive
	customer.updatedAt = at.UTC()
	return nil
}

func (customer *Customer) Deactivate(at time.Time) error {
	if at.IsZero() {
		return ErrDeactivationTimeRequired
	}

	customer.status = StatusInactive
	customer.updatedAt = at.UTC()
	return nil
}
