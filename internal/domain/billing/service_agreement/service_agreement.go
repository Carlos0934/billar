package serviceagreement

import (
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

type CreateParams struct {
	ID               billingvalues.ServiceAgreementID
	CustomerID       billingvalues.CustomerID
	Code             string
	Name             string
	Description      string
	BillingMode      BillingMode
	HourlyRateAmount int64
	Currency         billingvalues.CurrencyCode
	Validity         *billingvalues.DateRange
	CreatedAt        time.Time
}

type ServiceAgreement struct {
	id          billingvalues.ServiceAgreementID
	customerID  billingvalues.CustomerID
	code        string
	name        string
	description string
	billingMode BillingMode
	hourlyRate  billingvalues.Money
	validity    *billingvalues.DateRange
	active      bool
	createdAt   time.Time
	updatedAt   time.Time
}

func New(params CreateParams) (*ServiceAgreement, error) {
	if params.ID.IsZero() {
		return nil, ErrServiceAgreementIDRequired
	}
	if params.CustomerID.IsZero() {
		return nil, ErrCustomerIDRequired
	}
	if err := params.BillingMode.validate(); err != nil {
		return nil, err
	}
	if params.HourlyRateAmount <= 0 {
		return nil, ErrHourlyRateMustBePositive
	}
	if params.CreatedAt.IsZero() {
		return nil, ErrCreatedAtRequired
	}

	currency := params.Currency
	if currency.IsZero() {
		currency = billingvalues.DefaultCurrencyCode()
	}

	hourlyRate, err := billingvalues.NewMoney(params.HourlyRateAmount, currency)
	if err != nil {
		return nil, err
	}

	return &ServiceAgreement{
		id:          params.ID,
		customerID:  params.CustomerID,
		code:        strings.TrimSpace(params.Code),
		name:        strings.TrimSpace(params.Name),
		description: strings.TrimSpace(params.Description),
		billingMode: params.BillingMode,
		hourlyRate:  hourlyRate,
		validity:    params.Validity,
		active:      true,
		createdAt:   params.CreatedAt.UTC(),
		updatedAt:   params.CreatedAt.UTC(),
	}, nil
}

func (agreement *ServiceAgreement) ID() billingvalues.ServiceAgreementID {
	return agreement.id
}

func (agreement *ServiceAgreement) CustomerID() billingvalues.CustomerID {
	return agreement.customerID
}

func (agreement *ServiceAgreement) BillingMode() BillingMode {
	return agreement.billingMode
}

func (agreement *ServiceAgreement) HourlyRate() billingvalues.Money {
	return agreement.hourlyRate
}

func (agreement *ServiceAgreement) Currency() billingvalues.CurrencyCode {
	return agreement.hourlyRate.Currency()
}

func (agreement *ServiceAgreement) Validity() *billingvalues.DateRange {
	return agreement.validity
}

func (agreement *ServiceAgreement) CreatedAt() time.Time {
	return agreement.createdAt
}

func (agreement *ServiceAgreement) UpdatedAt() time.Time {
	return agreement.updatedAt
}

func (agreement *ServiceAgreement) Activate(at time.Time) error {
	if at.IsZero() {
		return ErrActivationTimeRequired
	}

	agreement.active = true
	agreement.updatedAt = at.UTC()
	return nil
}

func (agreement *ServiceAgreement) Deactivate(at time.Time) error {
	if at.IsZero() {
		return ErrDeactivationTimeRequired
	}

	agreement.active = false
	agreement.updatedAt = at.UTC()
	return nil
}

func (agreement *ServiceAgreement) ChangeHourlyRate(amount int64, at time.Time) error {
	if amount <= 0 {
		return ErrHourlyRateMustBePositive
	}
	if at.IsZero() {
		return ErrRateChangeTimeRequired
	}

	hourlyRate, err := billingvalues.NewMoney(amount, agreement.hourlyRate.Currency())
	if err != nil {
		return err
	}

	agreement.hourlyRate = hourlyRate
	agreement.updatedAt = at.UTC()
	return nil
}

func (agreement *ServiceAgreement) IsBillableOn(date time.Time) bool {
	if !agreement.active {
		return false
	}
	if agreement.validity == nil {
		return true
	}

	return agreement.validity.Contains(date)
}
