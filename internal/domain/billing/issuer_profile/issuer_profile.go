package issuerprofile

import (
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

type Identity struct {
	LegalName      string
	TaxID          billingvalues.TaxIdentifier
	BillingAddress billingvalues.Address
	Email          *billingvalues.EmailAddress
	Phone          *billingvalues.PhoneNumber
	Website        string
}

type InvoiceDefaults struct {
	DefaultCurrency     billingvalues.CurrencyCode
	InvoicePrefix       string
	PaymentInstructions string
}

type IssuerProfile struct {
	id                  billingvalues.IssuerProfileID
	legalName           string
	taxID               billingvalues.TaxIdentifier
	billingAddress      billingvalues.Address
	email               *billingvalues.EmailAddress
	phone               *billingvalues.PhoneNumber
	website             string
	defaultCurrency     billingvalues.CurrencyCode
	invoicePrefix       string
	paymentInstructions string
	createdAt           time.Time
	updatedAt           time.Time
}

func New(id billingvalues.IssuerProfileID, identity Identity, defaults InvoiceDefaults, createdAt time.Time) (*IssuerProfile, error) {
	if id.IsZero() {
		return nil, ErrIssuerProfileIDRequired
	}
	if createdAt.IsZero() {
		return nil, ErrCreatedAtRequired
	}

	normalizedIdentity, err := normalizeIdentity(identity)
	if err != nil {
		return nil, err
	}

	normalizedDefaults := normalizeDefaults(defaults)
	createdAt = createdAt.UTC()
	profile := &IssuerProfile{
		id:        id,
		createdAt: createdAt,
		updatedAt: createdAt,
	}
	profile.applyIdentity(normalizedIdentity)
	profile.applyDefaults(normalizedDefaults)

	return profile, nil
}

func (profile *IssuerProfile) UpdateIdentity(identity Identity, updatedAt time.Time) error {
	if updatedAt.IsZero() {
		return ErrUpdatedAtRequired
	}

	normalizedIdentity, err := normalizeIdentity(identity)
	if err != nil {
		return err
	}

	profile.applyIdentity(normalizedIdentity)
	profile.updatedAt = updatedAt.UTC()

	return nil
}

func (profile *IssuerProfile) ID() billingvalues.IssuerProfileID {
	return profile.id
}

func (profile *IssuerProfile) LegalName() string {
	return profile.legalName
}

func (profile *IssuerProfile) TaxID() billingvalues.TaxIdentifier {
	return profile.taxID
}

func (profile *IssuerProfile) BillingAddress() billingvalues.Address {
	return profile.billingAddress
}

func (profile *IssuerProfile) Email() *billingvalues.EmailAddress {
	return profile.email
}

func (profile *IssuerProfile) Phone() *billingvalues.PhoneNumber {
	return profile.phone
}

func (profile *IssuerProfile) Website() string {
	return profile.website
}

func (profile *IssuerProfile) DefaultCurrency() billingvalues.CurrencyCode {
	return profile.defaultCurrency
}

func (profile *IssuerProfile) InvoicePrefix() string {
	return profile.invoicePrefix
}

func (profile *IssuerProfile) PaymentInstructions() string {
	return profile.paymentInstructions
}

func (profile *IssuerProfile) CreatedAt() time.Time {
	return profile.createdAt
}

func (profile *IssuerProfile) UpdatedAt() time.Time {
	return profile.updatedAt
}

func normalizeIdentity(identity Identity) (Identity, error) {
	legalName := strings.TrimSpace(identity.LegalName)
	if legalName == "" {
		return Identity{}, ErrLegalNameRequired
	}
	if identity.TaxID.IsZero() {
		return Identity{}, ErrTaxIDRequired
	}
	if identity.BillingAddress.IsZero() {
		return Identity{}, ErrBillingAddressRequired
	}

	return Identity{
		LegalName:      legalName,
		TaxID:          identity.TaxID,
		BillingAddress: identity.BillingAddress,
		Email:          identity.Email,
		Phone:          identity.Phone,
		Website:        strings.TrimSpace(identity.Website),
	}, nil
}

func normalizeDefaults(defaults InvoiceDefaults) InvoiceDefaults {
	if defaults.DefaultCurrency.IsZero() {
		defaults.DefaultCurrency = billingvalues.DefaultCurrencyCode()
	}

	defaults.InvoicePrefix = strings.TrimSpace(defaults.InvoicePrefix)
	defaults.PaymentInstructions = strings.TrimSpace(defaults.PaymentInstructions)

	return defaults
}

func (profile *IssuerProfile) applyIdentity(identity Identity) {
	profile.legalName = identity.LegalName
	profile.taxID = identity.TaxID
	profile.billingAddress = identity.BillingAddress
	profile.email = identity.Email
	profile.phone = identity.Phone
	profile.website = identity.Website
}

func (profile *IssuerProfile) applyDefaults(defaults InvoiceDefaults) {
	profile.defaultCurrency = defaults.DefaultCurrency
	profile.invoicePrefix = defaults.InvoicePrefix
	profile.paymentInstructions = defaults.PaymentInstructions
}
