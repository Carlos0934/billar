package mcp

import (
	"encoding/json"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// -- Phase 1: BindArguments pointer semantics --

func TestBindArguments_AbsentFieldIsNil(t *testing.T) {
	t.Parallel()

	// A key that is absent from the map must leave the pointer field nil.
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id": "cus_123",
		// "email" intentionally omitted
	}

	var input CustomerProfileUpdateInput
	if err := req.BindArguments(&input); err != nil {
		t.Fatalf("BindArguments error = %v", err)
	}

	if input.ID != "cus_123" {
		t.Errorf("ID = %q, want %q", input.ID, "cus_123")
	}
	if input.Email != nil {
		t.Errorf("Email = %v, want nil (absent field must be nil pointer)", input.Email)
	}
}

func TestBindArguments_EmptyStringFieldIsNonNilPointer(t *testing.T) {
	t.Parallel()

	// A key present with empty string must produce a non-nil pointer to "".
	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id":    "cus_123",
		"email": "",
	}

	var input CustomerProfileUpdateInput
	if err := req.BindArguments(&input); err != nil {
		t.Fatalf("BindArguments error = %v", err)
	}

	if input.Email == nil {
		t.Fatal("Email = nil, want non-nil pointer to empty string (explicit clear)")
	}
	if *input.Email != "" {
		t.Errorf("*Email = %q, want %q", *input.Email, "")
	}
}

func TestBindArguments_PresentNonEmptyField(t *testing.T) {
	t.Parallel()

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id":    "cus_123",
		"email": "alice@example.com",
	}

	var input CustomerProfileUpdateInput
	if err := req.BindArguments(&input); err != nil {
		t.Fatalf("BindArguments error = %v", err)
	}

	if input.Email == nil {
		t.Fatal("Email = nil, want non-nil pointer")
	}
	if *input.Email != "alice@example.com" {
		t.Errorf("*Email = %q, want %q", *input.Email, "alice@example.com")
	}
}

// -- Phase 1: Nested AddressInput deserialization --

func TestBindArguments_AddressAbsentIsNilPointer(t *testing.T) {
	t.Parallel()

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id": "cus_123",
		// billing_address absent
	}

	var input CustomerProfileUpdateInput
	if err := req.BindArguments(&input); err != nil {
		t.Fatalf("BindArguments error = %v", err)
	}

	if input.BillingAddress != nil {
		t.Errorf("BillingAddress = %v, want nil (absent nested object)", input.BillingAddress)
	}
}

func TestBindArguments_AddressDeserializesIntoTypedStruct(t *testing.T) {
	t.Parallel()

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id": "cus_123",
		"billing_address": map[string]any{
			"street":      "123 Main St",
			"city":        "Santo Domingo",
			"state":       "DN",
			"postal_code": "10100",
			"country":     "DO",
		},
	}

	var input CustomerProfileUpdateInput
	if err := req.BindArguments(&input); err != nil {
		t.Fatalf("BindArguments error = %v", err)
	}

	if input.BillingAddress == nil {
		t.Fatal("BillingAddress = nil, want non-nil (address provided)")
	}
	if input.BillingAddress.Street != "123 Main St" {
		t.Errorf("Street = %q, want %q", input.BillingAddress.Street, "123 Main St")
	}
	if input.BillingAddress.City != "Santo Domingo" {
		t.Errorf("City = %q, want %q", input.BillingAddress.City, "Santo Domingo")
	}
	if input.BillingAddress.Country != "DO" {
		t.Errorf("Country = %q, want %q", input.BillingAddress.Country, "DO")
	}
	if input.BillingAddress.PostalCode != "10100" {
		t.Errorf("PostalCode = %q, want %q", input.BillingAddress.PostalCode, "10100")
	}
}

func TestBindArguments_PartialAddress(t *testing.T) {
	t.Parallel()

	req := mcplib.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"id": "cus_456",
		"billing_address": map[string]any{
			"country": "US",
			"city":    "New York",
			// street, state, postal_code absent — should be zero strings
		},
	}

	var input CustomerProfileUpdateInput
	if err := req.BindArguments(&input); err != nil {
		t.Fatalf("BindArguments error = %v", err)
	}

	if input.BillingAddress == nil {
		t.Fatal("BillingAddress = nil, want non-nil (partial address provided)")
	}
	if input.BillingAddress.Country != "US" {
		t.Errorf("Country = %q, want %q", input.BillingAddress.Country, "US")
	}
	if input.BillingAddress.City != "New York" {
		t.Errorf("City = %q, want %q", input.BillingAddress.City, "New York")
	}
	if input.BillingAddress.Street != "" {
		t.Errorf("Street = %q, want empty string (absent field in nested object)", input.BillingAddress.Street)
	}
}

// -- Phase 2: toCommand() converter methods --

func TestCustomerProfileCreateInput_ToCommand(t *testing.T) {
	t.Parallel()

	addr := &AddressInput{
		Street:     "1 Oak Ave",
		City:       "Miami",
		State:      "FL",
		PostalCode: "33101",
		Country:    "US",
	}

	input := CustomerProfileCreateInput{
		Type:            "company",
		LegalName:       "Acme Corp",
		TradeName:       "Acme",
		TaxID:           "123456789",
		Email:           "contact@acme.com",
		Phone:           "+1-800-555-0100",
		Website:         "https://acme.com",
		DefaultCurrency: "USD",
		Notes:           "VIP client",
		BillingAddress:  addr,
	}

	cmd := input.toCommand()

	if cmd.LegalEntityType != "company" {
		t.Errorf("LegalEntityType = %q, want %q", cmd.LegalEntityType, "company")
	}
	if cmd.LegalName != "Acme Corp" {
		t.Errorf("LegalName = %q, want %q", cmd.LegalName, "Acme Corp")
	}
	if cmd.DefaultCurrency != "USD" {
		t.Errorf("DefaultCurrency = %q, want %q", cmd.DefaultCurrency, "USD")
	}
	if cmd.Notes != "VIP client" {
		t.Errorf("Notes = %q, want %q", cmd.Notes, "VIP client")
	}
	if cmd.BillingAddress.Street != "1 Oak Ave" {
		t.Errorf("BillingAddress.Street = %q, want %q", cmd.BillingAddress.Street, "1 Oak Ave")
	}
	if cmd.BillingAddress.Country != "US" {
		t.Errorf("BillingAddress.Country = %q, want %q", cmd.BillingAddress.Country, "US")
	}
}

func TestCustomerProfileCreateInput_ToCommand_NoAddress(t *testing.T) {
	t.Parallel()

	input := CustomerProfileCreateInput{
		Type:            "individual",
		LegalName:       "John Doe",
		DefaultCurrency: "DOP",
	}

	cmd := input.toCommand()

	if cmd.LegalEntityType != "individual" {
		t.Errorf("LegalEntityType = %q, want %q", cmd.LegalEntityType, "individual")
	}
	// When BillingAddress is nil, the app AddressDTO should be zero value.
	zero := app.AddressDTO{}
	if cmd.BillingAddress != zero {
		t.Errorf("BillingAddress = %v, want zero value when input has nil address", cmd.BillingAddress)
	}
}

func TestCustomerProfileUpdateInput_ToCommand_AbsentFieldsAreNil(t *testing.T) {
	t.Parallel()

	input := CustomerProfileUpdateInput{
		ID: "cus_123",
		// All other fields absent
	}

	id, cmd := input.toCommand()

	if id != "cus_123" {
		t.Errorf("id = %q, want %q", id, "cus_123")
	}
	if cmd.Email != nil {
		t.Error("Email = non-nil, want nil (absent field)")
	}
	if cmd.DefaultCurrency != nil {
		t.Error("DefaultCurrency = non-nil, want nil (absent field)")
	}
	if cmd.BillingAddress != nil {
		t.Error("BillingAddress = non-nil, want nil (absent field)")
	}
}

func TestCustomerProfileUpdateInput_ToCommand_AddressMapped(t *testing.T) {
	t.Parallel()

	input := CustomerProfileUpdateInput{
		ID: "cus_123",
		BillingAddress: &AddressInput{
			Street:  "99 Elm St",
			City:    "Boston",
			Country: "US",
		},
	}

	_, cmd := input.toCommand()

	if cmd.BillingAddress == nil {
		t.Fatal("BillingAddress = nil, want non-nil pointer to AddressDTO")
	}
	if cmd.BillingAddress.Street != "99 Elm St" {
		t.Errorf("BillingAddress.Street = %q, want %q", cmd.BillingAddress.Street, "99 Elm St")
	}
	if cmd.BillingAddress.City != "Boston" {
		t.Errorf("BillingAddress.City = %q, want %q", cmd.BillingAddress.City, "Boston")
	}
}

func TestIssuerProfileCreateInput_ToCommand(t *testing.T) {
	t.Parallel()

	input := IssuerProfileCreateInput{
		Type:            "company",
		LegalName:       "My Billing Co",
		DefaultCurrency: "EUR",
		DefaultNotes:    "Thank you for your business",
		BillingAddress: &AddressInput{
			Country: "DE",
			City:    "Berlin",
		},
	}

	cmd := input.toCommand()

	if cmd.LegalEntityType != "company" {
		t.Errorf("LegalEntityType = %q, want %q", cmd.LegalEntityType, "company")
	}
	if cmd.DefaultNotes != "Thank you for your business" {
		t.Errorf("DefaultNotes = %q, want %q", cmd.DefaultNotes, "Thank you for your business")
	}
	if cmd.BillingAddress.Country != "DE" {
		t.Errorf("BillingAddress.Country = %q, want %q", cmd.BillingAddress.Country, "DE")
	}
}

func TestIssuerProfileUpdateInput_ToCommand_AbsentFieldsAreNil(t *testing.T) {
	t.Parallel()

	input := IssuerProfileUpdateInput{
		ID: "iss_456",
	}

	id, cmd := input.toCommand()

	if id != "iss_456" {
		t.Errorf("id = %q, want %q", id, "iss_456")
	}
	if cmd.DefaultCurrency != nil {
		t.Error("DefaultCurrency = non-nil, want nil (absent field)")
	}
	if cmd.LegalName != nil {
		t.Error("LegalName = non-nil, want nil (absent field)")
	}
}

func TestServiceAgreementCreateInput_ToCommand(t *testing.T) {
	t.Parallel()

	input := ServiceAgreementCreateInput{
		CustomerProfileID: "cus_789",
		Name:              "Premium Support",
		Description:       "24/7 support",
		BillingMode:       "hourly",
		HourlyRate:        25000,
		Currency:          "USD",
	}

	cmd := input.toCommand()

	if cmd.CustomerProfileID != "cus_789" {
		t.Errorf("CustomerProfileID = %q, want %q", cmd.CustomerProfileID, "cus_789")
	}
	if cmd.Name != "Premium Support" {
		t.Errorf("Name = %q, want %q", cmd.Name, "Premium Support")
	}
	if cmd.HourlyRate != 25000 {
		t.Errorf("HourlyRate = %d, want %d", cmd.HourlyRate, 25000)
	}
	if cmd.Currency != "USD" {
		t.Errorf("Currency = %q, want %q", cmd.Currency, "USD")
	}
}

func TestServiceAgreementUpdateRateInput_ToCommand(t *testing.T) {
	t.Parallel()

	input := ServiceAgreementUpdateRateInput{
		ID:         "sa_111",
		HourlyRate: 30000,
	}

	id, cmd := input.toCommand()

	if id != "sa_111" {
		t.Errorf("id = %q, want %q", id, "sa_111")
	}
	if cmd.HourlyRate != 30000 {
		t.Errorf("HourlyRate = %d, want %d", cmd.HourlyRate, 30000)
	}
}

// -- Ensure JSON round-trip for BindArguments works via json.Marshal + Unmarshal --

func TestCustomerProfileUpdateInput_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	// Simulate what BindArguments does internally: Marshal map → Unmarshal into struct.
	args := map[string]any{
		"id":               "cus_999",
		"default_currency": "DOP",
		// email absent
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("json.Marshal error = %v", err)
	}

	var input CustomerProfileUpdateInput
	if err := json.Unmarshal(data, &input); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}

	if input.DefaultCurrency == nil {
		t.Fatal("DefaultCurrency = nil, want non-nil (field was present)")
	}
	if *input.DefaultCurrency != "DOP" {
		t.Errorf("*DefaultCurrency = %q, want %q", *input.DefaultCurrency, "DOP")
	}
	if input.Email != nil {
		t.Errorf("Email = %v, want nil (absent field)", input.Email)
	}
}
