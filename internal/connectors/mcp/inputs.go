package mcp

import (
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/app"
)

// AddressInput is the connector-local typed struct for billing_address arguments.
// It mirrors app.AddressDTO but lives at the connector boundary for typed deserialization.
type AddressInput struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

// toDTO converts AddressInput to app.AddressDTO.
func (a AddressInput) toDTO() app.AddressDTO {
	return app.AddressDTO{
		Street:     a.Street,
		City:       a.City,
		State:      a.State,
		PostalCode: a.PostalCode,
		Country:    a.Country,
	}
}

// CustomerProfileCreateInput is the typed input struct for the customer_profile.create tool.
// Value fields are used because all required fields must be present; optional fields that
// are absent bind to zero strings, which is acceptable for create operations.
type CustomerProfileCreateInput struct {
	Type            string        `json:"type"`
	LegalName       string        `json:"legal_name"`
	TradeName       string        `json:"trade_name"`
	TaxID           string        `json:"tax_id"`
	Email           string        `json:"email"`
	Phone           string        `json:"phone"`
	Website         string        `json:"website"`
	DefaultCurrency string        `json:"default_currency"`
	Notes           string        `json:"notes"`
	BillingAddress  *AddressInput `json:"billing_address"`
}

// toCommand converts the input struct to app.CreateCustomerProfileCommand.
func (i CustomerProfileCreateInput) toCommand() app.CreateCustomerProfileCommand {
	cmd := app.CreateCustomerProfileCommand{
		LegalEntityType: strings.TrimSpace(i.Type),
		LegalName:       strings.TrimSpace(i.LegalName),
		TradeName:       strings.TrimSpace(i.TradeName),
		TaxID:           strings.TrimSpace(i.TaxID),
		Email:           strings.TrimSpace(i.Email),
		Phone:           strings.TrimSpace(i.Phone),
		Website:         strings.TrimSpace(i.Website),
		DefaultCurrency: strings.TrimSpace(i.DefaultCurrency),
		Notes:           strings.TrimSpace(i.Notes),
	}
	if i.BillingAddress != nil {
		cmd.BillingAddress = i.BillingAddress.toDTO()
	}
	return cmd
}

// CustomerProfileUpdateInput is the typed input struct for the customer_profile.update tool.
// Pointer fields distinguish absent (nil) from explicitly cleared ("").
type CustomerProfileUpdateInput struct {
	ID              string        `json:"id"`
	Type            *string       `json:"type"`
	LegalName       *string       `json:"legal_name"`
	TradeName       *string       `json:"trade_name"`
	TaxID           *string       `json:"tax_id"`
	Email           *string       `json:"email"`
	Phone           *string       `json:"phone"`
	Website         *string       `json:"website"`
	DefaultCurrency *string       `json:"default_currency"`
	Notes           *string       `json:"notes"`
	Status          *string       `json:"status"`
	BillingAddress  *AddressInput `json:"billing_address"`
}

// toCommand converts the input struct to (id, app.PatchCustomerProfileCommand).
func (i CustomerProfileUpdateInput) toCommand() (string, app.PatchCustomerProfileCommand) {
	var cmd app.PatchCustomerProfileCommand

	if i.Status != nil {
		s := strings.TrimSpace(*i.Status)
		cmd.Status = &s
	}
	if i.DefaultCurrency != nil {
		s := strings.TrimSpace(*i.DefaultCurrency)
		cmd.DefaultCurrency = &s
	}
	if i.Notes != nil {
		s := strings.TrimSpace(*i.Notes)
		cmd.Notes = &s
	}
	if i.Type != nil {
		s := strings.TrimSpace(*i.Type)
		cmd.LegalEntityType = &s
	}
	if i.LegalName != nil {
		s := strings.TrimSpace(*i.LegalName)
		cmd.LegalName = &s
	}
	if i.TradeName != nil {
		s := strings.TrimSpace(*i.TradeName)
		cmd.TradeName = &s
	}
	if i.TaxID != nil {
		s := strings.TrimSpace(*i.TaxID)
		cmd.TaxID = &s
	}
	if i.Email != nil {
		s := strings.TrimSpace(*i.Email)
		cmd.Email = &s
	}
	if i.Phone != nil {
		s := strings.TrimSpace(*i.Phone)
		cmd.Phone = &s
	}
	if i.Website != nil {
		s := strings.TrimSpace(*i.Website)
		cmd.Website = &s
	}
	if i.BillingAddress != nil {
		addr := i.BillingAddress.toDTO()
		cmd.BillingAddress = &addr
	}

	return strings.TrimSpace(i.ID), cmd
}

// IssuerProfileCreateInput is the typed input struct for the issuer_profile.create tool.
type IssuerProfileCreateInput struct {
	Type            string        `json:"type"`
	LegalName       string        `json:"legal_name"`
	TradeName       string        `json:"trade_name"`
	TaxID           string        `json:"tax_id"`
	Email           string        `json:"email"`
	Phone           string        `json:"phone"`
	Website         string        `json:"website"`
	DefaultCurrency string        `json:"default_currency"`
	DefaultNotes    string        `json:"default_notes"`
	BillingAddress  *AddressInput `json:"billing_address"`
}

// toCommand converts the input struct to app.CreateIssuerProfileCommand.
func (i IssuerProfileCreateInput) toCommand() app.CreateIssuerProfileCommand {
	cmd := app.CreateIssuerProfileCommand{
		LegalEntityType: strings.TrimSpace(i.Type),
		LegalName:       strings.TrimSpace(i.LegalName),
		TradeName:       strings.TrimSpace(i.TradeName),
		TaxID:           strings.TrimSpace(i.TaxID),
		Email:           strings.TrimSpace(i.Email),
		Phone:           strings.TrimSpace(i.Phone),
		Website:         strings.TrimSpace(i.Website),
		DefaultCurrency: strings.TrimSpace(i.DefaultCurrency),
		DefaultNotes:    strings.TrimSpace(i.DefaultNotes),
	}
	if i.BillingAddress != nil {
		cmd.BillingAddress = i.BillingAddress.toDTO()
	}
	return cmd
}

// IssuerProfileUpdateInput is the typed input struct for the issuer_profile.update tool.
// Pointer fields distinguish absent (nil) from explicitly cleared ("").
type IssuerProfileUpdateInput struct {
	ID              string        `json:"id"`
	Type            *string       `json:"type"`
	LegalName       *string       `json:"legal_name"`
	TradeName       *string       `json:"trade_name"`
	TaxID           *string       `json:"tax_id"`
	Email           *string       `json:"email"`
	Phone           *string       `json:"phone"`
	Website         *string       `json:"website"`
	DefaultCurrency *string       `json:"default_currency"`
	DefaultNotes    *string       `json:"default_notes"`
	BillingAddress  *AddressInput `json:"billing_address"`
}

// toCommand converts the input struct to (id, app.PatchIssuerProfileCommand).
func (i IssuerProfileUpdateInput) toCommand() (string, app.PatchIssuerProfileCommand) {
	var cmd app.PatchIssuerProfileCommand

	if i.DefaultCurrency != nil {
		s := strings.TrimSpace(*i.DefaultCurrency)
		cmd.DefaultCurrency = &s
	}
	if i.DefaultNotes != nil {
		s := strings.TrimSpace(*i.DefaultNotes)
		cmd.DefaultNotes = &s
	}
	if i.Type != nil {
		s := strings.TrimSpace(*i.Type)
		cmd.LegalEntityType = &s
	}
	if i.LegalName != nil {
		s := strings.TrimSpace(*i.LegalName)
		cmd.LegalName = &s
	}
	if i.TradeName != nil {
		s := strings.TrimSpace(*i.TradeName)
		cmd.TradeName = &s
	}
	if i.TaxID != nil {
		s := strings.TrimSpace(*i.TaxID)
		cmd.TaxID = &s
	}
	if i.Email != nil {
		s := strings.TrimSpace(*i.Email)
		cmd.Email = &s
	}
	if i.Phone != nil {
		s := strings.TrimSpace(*i.Phone)
		cmd.Phone = &s
	}
	if i.Website != nil {
		s := strings.TrimSpace(*i.Website)
		cmd.Website = &s
	}
	if i.BillingAddress != nil {
		addr := i.BillingAddress.toDTO()
		cmd.BillingAddress = &addr
	}

	return strings.TrimSpace(i.ID), cmd
}

// ServiceAgreementCreateInput is the typed input struct for the service_agreement.create tool.
type ServiceAgreementCreateInput struct {
	CustomerProfileID string `json:"customer_profile_id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	BillingMode       string `json:"billing_mode"`
	HourlyRate        int64  `json:"hourly_rate"`
	Currency          string `json:"currency"`
}

// toCommand converts the input struct to app.CreateServiceAgreementCommand.
func (i ServiceAgreementCreateInput) toCommand() app.CreateServiceAgreementCommand {
	return app.CreateServiceAgreementCommand{
		CustomerProfileID: strings.TrimSpace(i.CustomerProfileID),
		Name:              strings.TrimSpace(i.Name),
		Description:       strings.TrimSpace(i.Description),
		BillingMode:       strings.TrimSpace(i.BillingMode),
		HourlyRate:        i.HourlyRate,
		Currency:          strings.TrimSpace(i.Currency),
	}
}

// ServiceAgreementUpdateRateInput is the typed input struct for service_agreement.update_rate.
type ServiceAgreementUpdateRateInput struct {
	ID         string `json:"id"`
	HourlyRate int64  `json:"hourly_rate"`
}

// toCommand converts the input struct to (id, app.UpdateServiceAgreementRateCommand).
func (i ServiceAgreementUpdateRateInput) toCommand() (string, app.UpdateServiceAgreementRateCommand) {
	return strings.TrimSpace(i.ID), app.UpdateServiceAgreementRateCommand{
		HourlyRate: i.HourlyRate,
	}
}

// RecordTimeEntryInput is the typed input struct for the time_entry.record tool.
type RecordTimeEntryInput struct {
	CustomerProfileID  string    `json:"customer_profile_id"`
	ServiceAgreementID string    `json:"service_agreement_id"`
	Description        string    `json:"description"`
	Hours              int64     `json:"hours"`
	Billable           bool      `json:"billable"`
	Date               time.Time `json:"date"`
}

// toCommand converts the input struct to app.RecordTimeEntryCommand.
func (i RecordTimeEntryInput) toCommand() app.RecordTimeEntryCommand {
	return app.RecordTimeEntryCommand{
		CustomerProfileID:  strings.TrimSpace(i.CustomerProfileID),
		ServiceAgreementID: strings.TrimSpace(i.ServiceAgreementID),
		Description:        strings.TrimSpace(i.Description),
		Hours:              i.Hours,
		Billable:           i.Billable,
		Date:               i.Date,
	}
}

// UpdateTimeEntryInput is the typed input struct for the time_entry.update tool.
type UpdateTimeEntryInput struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Hours       int64  `json:"hours"`
}

// toCommand converts the input struct to app.UpdateTimeEntryCommand.
func (i UpdateTimeEntryInput) toCommand() app.UpdateTimeEntryCommand {
	return app.UpdateTimeEntryCommand{
		ID:          strings.TrimSpace(i.ID),
		Description: strings.TrimSpace(i.Description),
		Hours:       i.Hours,
	}
}
