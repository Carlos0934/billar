package issuerprofile

import "time"

func (profile *IssuerProfile) UpdateInvoiceDefaults(defaults InvoiceDefaults, updatedAt time.Time) error {
	if updatedAt.IsZero() {
		return ErrUpdatedAtRequired
	}

	normalizedDefaults := normalizeDefaults(defaults)
	profile.applyDefaults(normalizedDefaults)
	profile.updatedAt = updatedAt.UTC()

	return nil
}
