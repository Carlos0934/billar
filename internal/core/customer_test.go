package core

import (
	"strings"
	"testing"
	"time"
)

func TestCustomerType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  CustomerType
		want bool
	}{
		{name: "individual", got: CustomerTypeIndividual, want: true},
		{name: "company", got: CustomerTypeCompany, want: true},
		{name: "government", got: CustomerType("government"), want: false},
		{name: "empty", got: CustomerType(""), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.got.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomerStatus_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  CustomerStatus
		want bool
	}{
		{name: "active", got: CustomerStatusActive, want: true},
		{name: "inactive", got: CustomerStatusInactive, want: true},
		{name: "archived", got: CustomerStatus("archived"), want: false},
		{name: "empty", got: CustomerStatus(""), want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.got.IsValid(); got != tt.want {
				t.Fatalf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddress_Struct(t *testing.T) {
	t.Parallel()

	addr := Address{
		Street:     "123 Main St",
		City:       "Santo Domingo",
		PostalCode: "10101",
	}

	if addr.Street != "123 Main St" {
		t.Fatalf("Street = %q, want %q", addr.Street, "123 Main St")
	}
	if addr.City != "Santo Domingo" {
		t.Fatalf("City = %q, want %q", addr.City, "Santo Domingo")
	}
	if addr.PostalCode != "10101" {
		t.Fatalf("PostalCode = %q, want %q", addr.PostalCode, "10101")
	}
	if addr.State != "" {
		t.Fatalf("State = %q, want empty string", addr.State)
	}
	if addr.Country != "" {
		t.Fatalf("Country = %q, want empty string", addr.Country)
	}
}

func TestGenerateCustomerID(t *testing.T) {
	t.Parallel()

	id1 := generateCustomerID()
	id2 := generateCustomerID()

	if !strings.HasPrefix(id1, "cus_") {
		t.Fatalf("first id %q does not start with cus_", id1)
	}
	if !strings.HasPrefix(id2, "cus_") {
		t.Fatalf("second id %q does not start with cus_", id2)
	}
	if got, want := len(id1), len("cus_")+32; got != want {
		t.Fatalf("first id length = %d, want %d", got, want)
	}
	if got, want := len(id2), len("cus_")+32; got != want {
		t.Fatalf("second id length = %d, want %d", got, want)
	}
	if id1 == id2 {
		t.Fatal("expected distinct customer IDs")
	}
}

func TestNewCustomer_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params CustomerParams
	}{
		{
			name: "minimal required fields",
			params: CustomerParams{
				Type:      CustomerTypeIndividual,
				LegalName: "Maria Perez",
			},
		},
		{
			name: "optional fields preserved",
			params: CustomerParams{
				Type:           CustomerTypeCompany,
				LegalName:      "Acme SRL",
				TradeName:      "Acme",
				TaxID:          "001-1234567-8",
				Email:          "billing@acme.example",
				Phone:          "+1 809 555 0101",
				Website:        "https://acme.example",
				BillingAddress: Address{Street: "Calle 1", City: "Santo Domingo"},
				Notes:          "Preferred contact by email",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			customer, err := NewCustomer(tt.params)
			if err != nil {
				t.Fatalf("NewCustomer() error = %v", err)
			}

			if !strings.HasPrefix(customer.ID, "cus_") {
				t.Fatalf("ID = %q, want cus_ prefix", customer.ID)
			}
			if customer.Type != tt.params.Type {
				t.Fatalf("Type = %q, want %q", customer.Type, tt.params.Type)
			}
			if customer.LegalName != tt.params.LegalName {
				t.Fatalf("LegalName = %q, want %q", customer.LegalName, tt.params.LegalName)
			}
			if customer.TradeName != tt.params.TradeName {
				t.Fatalf("TradeName = %q, want %q", customer.TradeName, tt.params.TradeName)
			}
			if customer.TaxID != tt.params.TaxID {
				t.Fatalf("TaxID = %q, want %q", customer.TaxID, tt.params.TaxID)
			}
			if customer.Email != tt.params.Email {
				t.Fatalf("Email = %q, want %q", customer.Email, tt.params.Email)
			}
			if customer.Phone != tt.params.Phone {
				t.Fatalf("Phone = %q, want %q", customer.Phone, tt.params.Phone)
			}
			if customer.Website != tt.params.Website {
				t.Fatalf("Website = %q, want %q", customer.Website, tt.params.Website)
			}
			if customer.BillingAddress != tt.params.BillingAddress {
				t.Fatalf("BillingAddress = %#v, want %#v", customer.BillingAddress, tt.params.BillingAddress)
			}
			if customer.Status != CustomerStatusActive {
				t.Fatalf("Status = %q, want %q", customer.Status, CustomerStatusActive)
			}
			if customer.DefaultCurrency != "USD" {
				t.Fatalf("DefaultCurrency = %q, want %q", customer.DefaultCurrency, "USD")
			}
			if customer.Notes != tt.params.Notes {
				t.Fatalf("Notes = %q, want %q", customer.Notes, tt.params.Notes)
			}
			if customer.CreatedAt.IsZero() {
				t.Fatal("CreatedAt is zero")
			}
			if customer.UpdatedAt.IsZero() {
				t.Fatal("UpdatedAt is zero")
			}
			if !customer.CreatedAt.Equal(customer.UpdatedAt) {
				t.Fatalf("CreatedAt = %v, UpdatedAt = %v, want equal", customer.CreatedAt, customer.UpdatedAt)
			}
		})
	}
}

func TestNewCustomer_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  CustomerParams
		wantErr string
	}{
		{
			name: "blank legal name",
			params: CustomerParams{
				Type:      CustomerTypeIndividual,
				LegalName: "   ",
			},
			wantErr: "legal name",
		},
		{
			name: "invalid customer type",
			params: CustomerParams{
				Type:      CustomerType("government"),
				LegalName: "Maria Perez",
			},
			wantErr: "type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewCustomer(tt.params)
			if err == nil {
				t.Fatal("NewCustomer() error = nil, want non-nil")
			}
			if tt.wantErr != "" && !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestCanReceiveInvoices(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		customer Customer
		want     bool
	}{
		{name: "active", customer: Customer{Status: CustomerStatusActive}, want: true},
		{name: "inactive", customer: Customer{Status: CustomerStatusInactive}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.customer.CanReceiveInvoices(); got != tt.want {
				t.Fatalf("CanReceiveInvoices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCustomerPatch_ApplyPatch_NilPointer_SkipsField(t *testing.T) {
	t.Parallel()

	// Setup: existing customer with all fields populated
	original := Customer{
		ID:              "cus_test123",
		Type:            CustomerTypeCompany,
		LegalName:       "Original Legal Name",
		TradeName:       "Original Trade Name",
		TaxID:           "Original Tax ID",
		Email:           "original@example.com",
		Phone:           "+1 555 0100",
		Website:         "https://original.example",
		BillingAddress:  Address{Street: "Original St", City: "Original City"},
		Notes:           "Original notes",
		DefaultCurrency: "EUR",
		UpdatedAt:       time.Now().Add(-24 * time.Hour), // Old timestamp
	}

	// Patch with all nil pointers (no fields to update)
	patch := CustomerPatch{}

	customer := original
	beforeApply := customer.UpdatedAt
	customer.ApplyPatch(patch)

	// All fields should remain unchanged
	if customer.Type != original.Type {
		t.Fatalf("Type = %q, want %q (nil pointer should skip)", customer.Type, original.Type)
	}
	if customer.LegalName != original.LegalName {
		t.Fatalf("LegalName = %q, want %q (nil pointer should skip)", customer.LegalName, original.LegalName)
	}
	if customer.TradeName != original.TradeName {
		t.Fatalf("TradeName = %q, want %q (nil pointer should skip)", customer.TradeName, original.TradeName)
	}
	if customer.TaxID != original.TaxID {
		t.Fatalf("TaxID = %q, want %q (nil pointer should skip)", customer.TaxID, original.TaxID)
	}
	if customer.Email != original.Email {
		t.Fatalf("Email = %q, want %q (nil pointer should skip)", customer.Email, original.Email)
	}
	if customer.Phone != original.Phone {
		t.Fatalf("Phone = %q, want %q (nil pointer should skip)", customer.Phone, original.Phone)
	}
	if customer.Website != original.Website {
		t.Fatalf("Website = %q, want %q (nil pointer should skip)", customer.Website, original.Website)
	}
	if customer.BillingAddress != original.BillingAddress {
		t.Fatalf("BillingAddress = %#v, want %#v (nil pointer should skip)", customer.BillingAddress, original.BillingAddress)
	}
	if customer.Notes != original.Notes {
		t.Fatalf("Notes = %q, want %q (nil pointer should skip)", customer.Notes, original.Notes)
	}
	if customer.DefaultCurrency != original.DefaultCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q (nil pointer should skip)", customer.DefaultCurrency, original.DefaultCurrency)
	}
	// UpdatedAt should not change when nothing is patched
	if !customer.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should not change when no fields are patched")
	}
}

func TestCustomerPatch_ApplyPatch_EmptyString_ClearsField(t *testing.T) {
	t.Parallel()

	// Setup: existing customer with all fields populated
	original := Customer{
		ID:              "cus_test123",
		Type:            CustomerTypeCompany,
		LegalName:       "Original Legal Name",
		TradeName:       "Original Trade Name",
		TaxID:           "Original Tax ID",
		Email:           "original@example.com",
		Phone:           "+1 555 0100",
		Website:         "https://original.example",
		BillingAddress:  Address{Street: "Original St", City: "Original City"},
		Notes:           "Original notes",
		DefaultCurrency: "EUR",
	}

	// Patch with non-nil empty strings (clear fields)
	emptyStr := ""
	patch := CustomerPatch{
		TradeName:       &emptyStr,
		TaxID:           &emptyStr,
		Email:           &emptyStr,
		Phone:           &emptyStr,
		Website:         &emptyStr,
		Notes:           &emptyStr,
		DefaultCurrency: nil, // Skip this one
	}

	customer := original
	beforeApply := customer.UpdatedAt
	customer.ApplyPatch(patch)

	// Fields with empty string should be cleared
	if customer.TradeName != "" {
		t.Fatalf("TradeName = %q, want empty string (cleared)", customer.TradeName)
	}
	if customer.TaxID != "" {
		t.Fatalf("TaxID = %q, want empty string (cleared)", customer.TaxID)
	}
	if customer.Email != "" {
		t.Fatalf("Email = %q, want empty string (cleared)", customer.Email)
	}
	if customer.Phone != "" {
		t.Fatalf("Phone = %q, want empty string (cleared)", customer.Phone)
	}
	if customer.Website != "" {
		t.Fatalf("Website = %q, want empty string (cleared)", customer.Website)
	}
	if customer.Notes != "" {
		t.Fatalf("Notes = %q, want empty string (cleared)", customer.Notes)
	}

	// Nil pointer fields should remain unchanged
	if customer.LegalName != original.LegalName {
		t.Fatalf("LegalName = %q, want %q (nil pointer should skip)", customer.LegalName, original.LegalName)
	}
	if customer.DefaultCurrency != original.DefaultCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q (nil pointer should skip)", customer.DefaultCurrency, original.DefaultCurrency)
	}

	// UpdatedAt should be updated when fields are patched
	if customer.UpdatedAt.Before(beforeApply) || customer.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should be more recent after patching")
	}
}

func TestCustomerPatch_ApplyPatch_NonNilValue_ReplacesField(t *testing.T) {
	t.Parallel()

	// Setup: existing customer with original values
	original := Customer{
		ID:              "cus_test123",
		Type:            CustomerTypeIndividual,
		LegalName:       "Original Legal Name",
		TradeName:       "Original Trade Name",
		TaxID:           "Original Tax ID",
		Email:           "original@example.com",
		Phone:           "+1 555 0100",
		Website:         "https://original.example",
		BillingAddress:  Address{Street: "Original St", City: "Original City"},
		Notes:           "Original notes",
		DefaultCurrency: "USD",
	}

	// Patch with non-nil values (replace fields)
	newType := CustomerTypeCompany
	newLegalName := "New Legal Name"
	newTradeName := "New Trade Name"
	newTaxID := "New Tax ID"
	newEmail := "new@example.com"
	newPhone := "+1 555 9999"
	newWebsite := "https://new.example"
	newAddress := Address{Street: "New St", City: "New City", Country: "New Country"}
	newNotes := "New notes"
	newCurrency := "EUR"

	patch := CustomerPatch{
		Type:            &newType,
		LegalName:       &newLegalName,
		TradeName:       &newTradeName,
		TaxID:           &newTaxID,
		Email:           &newEmail,
		Phone:           &newPhone,
		Website:         &newWebsite,
		BillingAddress:  &newAddress,
		Notes:           &newNotes,
		DefaultCurrency: &newCurrency,
	}

	customer := original
	beforeApply := customer.UpdatedAt
	customer.ApplyPatch(patch)

	// All fields should be replaced
	if customer.Type != newType {
		t.Fatalf("Type = %q, want %q", customer.Type, newType)
	}
	if customer.LegalName != newLegalName {
		t.Fatalf("LegalName = %q, want %q", customer.LegalName, newLegalName)
	}
	if customer.TradeName != newTradeName {
		t.Fatalf("TradeName = %q, want %q", customer.TradeName, newTradeName)
	}
	if customer.TaxID != newTaxID {
		t.Fatalf("TaxID = %q, want %q", customer.TaxID, newTaxID)
	}
	if customer.Email != newEmail {
		t.Fatalf("Email = %q, want %q", customer.Email, newEmail)
	}
	if customer.Phone != newPhone {
		t.Fatalf("Phone = %q, want %q", customer.Phone, newPhone)
	}
	if customer.Website != newWebsite {
		t.Fatalf("Website = %q, want %q", customer.Website, newWebsite)
	}
	if customer.BillingAddress != newAddress {
		t.Fatalf("BillingAddress = %#v, want %#v", customer.BillingAddress, newAddress)
	}
	if customer.Notes != newNotes {
		t.Fatalf("Notes = %q, want %q", customer.Notes, newNotes)
	}
	if customer.DefaultCurrency != newCurrency {
		t.Fatalf("DefaultCurrency = %q, want %q", customer.DefaultCurrency, newCurrency)
	}

	// Unchangeable fields should remain unchanged
	if customer.ID != original.ID {
		t.Fatalf("ID = %q, want %q (ID should not change)", customer.ID, original.ID)
	}
	if customer.Status != original.Status {
		t.Fatalf("Status = %q, want %q (Status should not change via patch)", customer.Status, original.Status)
	}
	if customer.CreatedAt != original.CreatedAt {
		t.Fatal("CreatedAt should not change")
	}

	// UpdatedAt should be updated
	if customer.UpdatedAt.Before(beforeApply) || customer.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should be more recent after patching")
	}
}

func TestCustomerPatch_ApplyPatch_EmptyAddress_ClearsAllAddressFields(t *testing.T) {
	t.Parallel()

	// Setup: existing customer with populated address
	original := Customer{
		ID: "cus_test123",
		BillingAddress: Address{
			Street:     "Original St",
			City:       "Original City",
			State:      "Original State",
			PostalCode: "12345",
			Country:    "Original Country",
		},
	}

	// Patch with empty address (clear all address fields)
	emptyAddress := Address{}
	patch := CustomerPatch{
		BillingAddress: &emptyAddress,
	}

	customer := original
	customer.ApplyPatch(patch)

	// All address fields should be empty
	if customer.BillingAddress.Street != "" {
		t.Fatalf("BillingAddress.Street = %q, want empty", customer.BillingAddress.Street)
	}
	if customer.BillingAddress.City != "" {
		t.Fatalf("BillingAddress.City = %q, want empty", customer.BillingAddress.City)
	}
	if customer.BillingAddress.State != "" {
		t.Fatalf("BillingAddress.State = %q, want empty", customer.BillingAddress.State)
	}
	if customer.BillingAddress.PostalCode != "" {
		t.Fatalf("BillingAddress.PostalCode = %q, want empty", customer.BillingAddress.PostalCode)
	}
	if customer.BillingAddress.Country != "" {
		t.Fatalf("BillingAddress.Country = %q, want empty", customer.BillingAddress.Country)
	}
}

func TestCustomerPatch_ApplyPatch_PartialAddress_UpdatesSpecifiedFields(t *testing.T) {
	t.Parallel()

	// Setup: existing customer with partial address
	original := Customer{
		ID: "cus_test123",
		BillingAddress: Address{
			Street:     "Original St",
			City:       "Original City",
			State:      "Original State",
			PostalCode: "12345",
			Country:    "Original Country",
		},
	}

	// Patch with new address - only Street and City changed
	newAddress := Address{
		Street: "New St",
		City:   "New City",
		// State, PostalCode, Country not provided in patch
	}
	patch := CustomerPatch{
		BillingAddress: &newAddress,
	}

	customer := original
	customer.ApplyPatch(patch)

	// Address should be fully replaced with new values
	// (State, PostalCode, Country are empty in new address)
	if customer.BillingAddress.Street != "New St" {
		t.Fatalf("BillingAddress.Street = %q, want New St", customer.BillingAddress.Street)
	}
	if customer.BillingAddress.City != "New City" {
		t.Fatalf("BillingAddress.City = %q, want New City", customer.BillingAddress.City)
	}
	if customer.BillingAddress.State != "" {
		t.Fatalf("BillingAddress.State = %q, want empty (address was replaced)", customer.BillingAddress.State)
	}
	if customer.BillingAddress.PostalCode != "" {
		t.Fatalf("BillingAddress.PostalCode = %q, want empty (address was replaced)", customer.BillingAddress.PostalCode)
	}
	if customer.BillingAddress.Country != "" {
		t.Fatalf("BillingAddress.Country = %q, want empty (address was replaced)", customer.BillingAddress.Country)
	}
}

func TestCustomer_ValidateDelete_ReturnsNil(t *testing.T) {
	t.Parallel()

	// ValidateDelete is a seam for future relationship-protection logic.
	// For now, it always returns nil (no blocking conditions).
	customer := Customer{
		ID:        "cus_test123",
		Type:      CustomerTypeCompany,
		LegalName: "Test Customer",
		Status:    CustomerStatusActive,
	}

	err := customer.ValidateDelete()
	if err != nil {
		t.Fatalf("ValidateDelete() = %v, want nil", err)
	}
}

func TestCustomer_Validate_ReturnsErrorForInvalidState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		customer Customer
		wantErr  string
	}{
		{
			name: "blank legal name after patch",
			customer: Customer{
				ID:        "cus_test123",
				Type:      CustomerTypeCompany,
				LegalName: "", // Invalid: blank
			},
			wantErr: "legal name",
		},
		{
			name: "invalid type after patch",
			customer: Customer{
				ID:        "cus_test123",
				Type:      CustomerType("invalid"),
				LegalName: "Test Customer",
			},
			wantErr: "invalid customer type",
		},
		{
			name: "valid customer",
			customer: Customer{
				ID:        "cus_test123",
				Type:      CustomerTypeCompany,
				LegalName: "Test Customer",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.customer.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Fatal("Validate() = nil, want non-nil")
				}
				if !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
					t.Fatalf("Validate() error = %q, want contains %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}
