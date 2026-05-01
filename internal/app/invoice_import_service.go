package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/core"
)

const importInvoiceSchemaV1 = "billar.invoice.import/v1"

func (s InvoiceService) ImportIssued(ctx context.Context, cmd ImportIssuedInvoiceCommand) (InvoiceDTO, error) {
	if s.invoices == nil || s.profiles == nil {
		return InvoiceDTO{}, errors.New("invoice service dependencies are required")
	}
	importer, ok := s.invoices.(invoiceImporter)
	if !ok {
		return InvoiceDTO{}, errors.New("invoice import store is required")
	}
	payload := cmd.Payload
	if strings.TrimSpace(payload.Schema) != importInvoiceSchemaV1 {
		return InvoiceDTO{}, ErrUnsupportedImportSchema
	}
	customer, err := s.resolveImportCustomer(ctx, payload.Customer)
	if err != nil {
		return InvoiceDTO{}, err
	}
	if _, err := s.resolveImportIssuer(ctx, payload.Issuer); err != nil {
		return InvoiceDTO{}, err
	}
	invoiceDate, err := parseRequiredImportDate(payload.InvoiceDate, "invoice_date")
	if err != nil {
		return InvoiceDTO{}, err
	}
	dueDate, err := parseRequiredImportDate(payload.DueDate, "due_date")
	if err != nil {
		return InvoiceDTO{}, err
	}
	importedAt, err := parseOptionalImportTime(payload.Source.ImportedAt)
	if err != nil {
		return InvoiceDTO{}, err
	}

	lines := make([]core.InvoiceLine, 0, len(payload.Lines))
	for _, in := range payload.Lines {
		line, err := core.NewImportedInvoiceLine(core.ImportInvoiceLineParams{Description: in.Description, AmountMinor: in.AmountMinor, TaxMinor: in.TaxMinor, QuantityDisplay: in.QuantityDisplay, UnitPriceDisplay: in.UnitPriceDisplay, Currency: payload.Currency})
		if err != nil {
			return InvoiceDTO{}, fmt.Errorf("import invoice line: %w", err)
		}
		line.ServiceAgreementID = strings.TrimSpace(in.ServiceAgreementID)
		lines = append(lines, line)
	}
	invoice, err := core.NewImportedInvoice(core.ImportInvoiceParams{CustomerID: customer.ID, InvoiceNumber: payload.InvoiceNumber, InvoiceDate: invoiceDate, DueDate: dueDate, Currency: payload.Currency, PaymentTerms: payload.PaymentTerms, PaymentCommunication: payload.PaymentCommunication, ImportSource: payload.Source.System, ExternalNumber: payload.Source.ExternalID, ImportedAt: importedAt, Lines: lines, SubtotalMinor: payload.Totals.SubtotalMinor, TaxTotalMinor: payload.Totals.TaxTotalMinor, GrandTotalMinor: payload.Totals.GrandTotalMinor})
	if err != nil {
		if strings.Contains(err.Error(), "totals do not balance") {
			return InvoiceDTO{}, ErrImportTotalsMismatch
		}
		return InvoiceDTO{}, fmt.Errorf("import invoice: %w", err)
	}
	if err := importer.ImportIssued(ctx, &invoice); err != nil {
		return InvoiceDTO{}, fmt.Errorf("import invoice: %w", err)
	}
	return invoiceToDTO(invoice, nil), nil
}

func (s InvoiceService) resolveImportCustomer(ctx context.Context, customer ImportPayloadCustomer) (*core.CustomerProfile, error) {
	if id := strings.TrimSpace(customer.CustomerProfileID); id != "" {
		profile, err := s.profiles.GetByID(ctx, id)
		if err != nil {
			return nil, ErrImportCustomerProfileNotFound
		}
		if profile == nil || !profile.CanReceiveInvoices() {
			return nil, ErrImportCustomerProfileNotFound
		}
		return profile, nil
	}
	if s.legalEntities == nil {
		return nil, ErrImportCustomerProfileNotFound
	}
	var entity *core.LegalEntity
	var err error
	if taxID := strings.TrimSpace(customer.TaxID); taxID != "" {
		entity, err = s.legalEntities.FindByTaxID(ctx, taxID)
	} else if legalName := strings.TrimSpace(customer.LegalName); legalName != "" {
		entity, err = s.legalEntities.FindByLegalName(ctx, legalName)
	} else {
		return nil, ErrImportCustomerProfileNotFound
	}
	if err != nil || entity == nil {
		return nil, ErrImportCustomerProfileNotFound
	}
	profileStore, ok := s.profiles.(customerProfileByLegalEntityID)
	if !ok {
		return nil, ErrImportCustomerProfileNotFound
	}
	profile, err := profileStore.GetByLegalEntityID(ctx, entity.ID)
	if err != nil || profile == nil || !profile.CanReceiveInvoices() {
		return nil, ErrImportCustomerProfileNotFound
	}
	return profile, nil
}

func (s InvoiceService) resolveImportIssuer(ctx context.Context, issuer ImportPayloadIssuer) (*core.IssuerProfile, error) {
	if s.issuers == nil {
		return nil, errors.New("issuer profile store is required")
	}
	if id := strings.TrimSpace(issuer.IssuerProfileID); id != "" {
		return s.issuers.GetByID(ctx, id)
	}
	return s.issuers.GetDefault(ctx)
}

func parseRequiredImportDate(value, field string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must be YYYY-MM-DD", field)
	}
	return t.UTC(), nil
}

func parseOptionalImportTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("imported_at must be RFC3339")
	}
	return t.UTC(), nil
}
