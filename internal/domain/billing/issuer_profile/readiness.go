package issuerprofile

type MissingRequirement string

const (
	MissingLegalName           MissingRequirement = "legal_name"
	MissingTaxID               MissingRequirement = "tax_id"
	MissingBillingAddress      MissingRequirement = "billing_address"
	MissingDirectContact       MissingRequirement = "direct_contact"
	MissingDefaultCurrency     MissingRequirement = "default_currency"
	MissingInvoicePrefix       MissingRequirement = "invoice_prefix"
	MissingPaymentInstructions MissingRequirement = "payment_instructions"
)

type IssuanceReadiness struct {
	Ready   bool
	Missing []MissingRequirement
}

func (profile *IssuerProfile) EvaluateIssuanceReadiness() IssuanceReadiness {
	missing := make([]MissingRequirement, 0, 7)

	if profile == nil || profile.legalName == "" {
		missing = append(missing, MissingLegalName)
	}
	if profile == nil || profile.taxID.IsZero() {
		missing = append(missing, MissingTaxID)
	}
	if profile == nil || profile.billingAddress.IsZero() {
		missing = append(missing, MissingBillingAddress)
	}
	if profile == nil || !profile.hasDirectContact() {
		missing = append(missing, MissingDirectContact)
	}
	if profile == nil || profile.defaultCurrency.IsZero() {
		missing = append(missing, MissingDefaultCurrency)
	}
	if profile == nil || profile.invoicePrefix == "" {
		missing = append(missing, MissingInvoicePrefix)
	}
	if profile == nil || profile.paymentInstructions == "" {
		missing = append(missing, MissingPaymentInstructions)
	}

	return IssuanceReadiness{
		Ready:   len(missing) == 0,
		Missing: missing,
	}
}

func (profile *IssuerProfile) hasDirectContact() bool {
	if profile == nil {
		return false
	}

	return (profile.email != nil && !profile.email.IsZero()) ||
		(profile.phone != nil && !profile.phone.IsZero())
}
