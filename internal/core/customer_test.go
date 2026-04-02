package core

import (
	"strings"
	"testing"
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
