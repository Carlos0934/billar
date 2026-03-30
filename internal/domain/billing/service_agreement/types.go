package serviceagreement

type BillingMode string

const ModeHourly BillingMode = "hourly"

func (billingMode BillingMode) validate() error {
	if billingMode != ModeHourly {
		return ErrBillingModeMustBeHourly
	}

	return nil
}
