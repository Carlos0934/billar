package core

import (
	"strings"
	"testing"
	"time"
)

func TestEntityType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  EntityType
		want bool
	}{
		{name: "company", got: EntityTypeCompany, want: true},
		{name: "individual", got: EntityTypeIndividual, want: true},
		{name: "invalid", got: EntityType("invalid"), want: false},
		{name: "empty", got: EntityType(""), want: false},
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

func TestLegalEntityIDPrefix(t *testing.T) {
	t.Parallel()

	// Test that generated IDs have the correct prefix
	id := generateLegalEntityID()
	if !strings.HasPrefix(id, "le_") {
		t.Fatalf("ID = %q, want le_ prefix", id)
	}
	if got, want := len(id), len("le_")+32; got != want {
		t.Fatalf("ID length = %d, want %d", got, want)
	}
}

func TestLegalEntityIDUniqueness(t *testing.T) {
	t.Parallel()

	id1 := generateLegalEntityID()
	id2 := generateLegalEntityID()

	if id1 == id2 {
		t.Fatal("expected distinct legal entity IDs")
	}
}

func TestNewLegalEntity_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params LegalEntityParams
	}{
		{
			name: "minimal required fields - company",
			params: LegalEntityParams{
				Type:      EntityTypeCompany,
				LegalName: "Acme Corporation S.A.",
			},
		},
		{
			name: "minimal required fields - individual",
			params: LegalEntityParams{
				Type:      EntityTypeIndividual,
				LegalName: "Maria Perez",
			},
		},
		{
			name: "all fields populated",
			params: LegalEntityParams{
				Type:           EntityTypeCompany,
				LegalName:      "Acme Corporation S.A.",
				TradeName:      "Acme",
				TaxID:          "RNC-123456789",
				Email:          "contact@acme.example",
				Phone:          "+1 809 555 0101",
				Website:        "https://acme.example",
				BillingAddress: Address{Street: "123 Main St", City: "Santo Domingo", Country: "DO"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			le, err := NewLegalEntity(tt.params)
			if err != nil {
				t.Fatalf("NewLegalEntity() error = %v", err)
			}

			if !strings.HasPrefix(le.ID, "le_") {
				t.Fatalf("ID = %q, want le_ prefix", le.ID)
			}
			if le.Type != tt.params.Type {
				t.Fatalf("Type = %q, want %q", le.Type, tt.params.Type)
			}
			if le.LegalName != tt.params.LegalName {
				t.Fatalf("LegalName = %q, want %q", le.LegalName, tt.params.LegalName)
			}
			if le.TradeName != tt.params.TradeName {
				t.Fatalf("TradeName = %q, want %q", le.TradeName, tt.params.TradeName)
			}
			if le.TaxID != tt.params.TaxID {
				t.Fatalf("TaxID = %q, want %q", le.TaxID, tt.params.TaxID)
			}
			if le.Email != tt.params.Email {
				t.Fatalf("Email = %q, want %q", le.Email, tt.params.Email)
			}
			if le.Phone != tt.params.Phone {
				t.Fatalf("Phone = %q, want %q", le.Phone, tt.params.Phone)
			}
			if le.Website != tt.params.Website {
				t.Fatalf("Website = %q, want %q", le.Website, tt.params.Website)
			}
			if le.BillingAddress != tt.params.BillingAddress {
				t.Fatalf("BillingAddress = %#v, want %#v", le.BillingAddress, tt.params.BillingAddress)
			}
			if le.CreatedAt.IsZero() {
				t.Fatal("CreatedAt is zero")
			}
			if le.UpdatedAt.IsZero() {
				t.Fatal("UpdatedAt is zero")
			}
			if !le.CreatedAt.Equal(le.UpdatedAt) {
				t.Fatalf("CreatedAt = %v, UpdatedAt = %v, want equal", le.CreatedAt, le.UpdatedAt)
			}
		})
	}
}

func TestNewLegalEntity_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  LegalEntityParams
		wantErr string
	}{
		{
			name: "blank legal name",
			params: LegalEntityParams{
				Type:      EntityTypeCompany,
				LegalName: "   ",
			},
			wantErr: "legal name",
		},
		{
			name: "invalid entity type",
			params: LegalEntityParams{
				Type:      EntityType("government"),
				LegalName: "Maria Perez",
			},
			wantErr: "type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewLegalEntity(tt.params)
			if err == nil {
				t.Fatal("NewLegalEntity() error = nil, want non-nil")
			}
			if tt.wantErr != "" && !strings.Contains(strings.ToLower(err.Error()), tt.wantErr) {
				t.Fatalf("error = %q, want contains %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestLegalEntity_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		entity  LegalEntity
		wantErr string
	}{
		{
			name: "valid entity",
			entity: LegalEntity{
				ID:        "le_test123",
				Type:      EntityTypeCompany,
				LegalName: "Acme Corporation",
			},
			wantErr: "",
		},
		{
			name: "blank legal name",
			entity: LegalEntity{
				ID:        "le_test123",
				Type:      EntityTypeCompany,
				LegalName: "",
			},
			wantErr: "legal name",
		},
		{
			name: "invalid type",
			entity: LegalEntity{
				ID:        "le_test123",
				Type:      EntityType("invalid"),
				LegalName: "Acme Corporation",
			},
			wantErr: "invalid entity type",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.entity.Validate()
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

func TestLegalEntityPatch_ApplyPatch_NilPointer_SkipsField(t *testing.T) {
	t.Parallel()

	original := LegalEntity{
		ID:             "le_test123",
		Type:           EntityTypeCompany,
		LegalName:      "Original Legal Name",
		TradeName:      "Original Trade Name",
		TaxID:          "Original Tax ID",
		Email:          "original@example.com",
		Phone:          "+1 555 0100",
		Website:        "https://original.example",
		BillingAddress: Address{Street: "Original St", City: "Original City"},
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	patch := LegalEntityPatch{}
	entity := original
	beforeApply := entity.UpdatedAt
	entity.ApplyPatch(patch)

	// All fields should remain unchanged
	if entity.Type != original.Type {
		t.Fatalf("Type = %q, want %q (nil pointer should skip)", entity.Type, original.Type)
	}
	if entity.LegalName != original.LegalName {
		t.Fatalf("LegalName = %q, want %q (nil pointer should skip)", entity.LegalName, original.LegalName)
	}
	if entity.TradeName != original.TradeName {
		t.Fatalf("TradeName = %q, want %q (nil pointer should skip)", entity.TradeName, original.TradeName)
	}
	if entity.TaxID != original.TaxID {
		t.Fatalf("TaxID = %q, want %q (nil pointer should skip)", entity.TaxID, original.TaxID)
	}
	if entity.Email != original.Email {
		t.Fatalf("Email = %q, want %q (nil pointer should skip)", entity.Email, original.Email)
	}
	if entity.Phone != original.Phone {
		t.Fatalf("Phone = %q, want %q (nil pointer should skip)", entity.Phone, original.Phone)
	}
	if entity.Website != original.Website {
		t.Fatalf("Website = %q, want %q (nil pointer should skip)", entity.Website, original.Website)
	}
	if entity.BillingAddress != original.BillingAddress {
		t.Fatalf("BillingAddress = %#v, want %#v (nil pointer should skip)", entity.BillingAddress, original.BillingAddress)
	}
	// UpdatedAt should not change when nothing is patched
	if !entity.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should not change when no fields are patched")
	}
}

func TestLegalEntityPatch_ApplyPatch_NonNilValue_ReplacesField(t *testing.T) {
	t.Parallel()

	original := LegalEntity{
		ID:             "le_test123",
		Type:           EntityTypeIndividual,
		LegalName:      "Original Legal Name",
		TradeName:      "Original Trade Name",
		TaxID:          "Original Tax ID",
		Email:          "original@example.com",
		Phone:          "+1 555 0100",
		Website:        "https://original.example",
		BillingAddress: Address{Street: "Original St", City: "Original City"},
	}

	newType := EntityTypeCompany
	newLegalName := "New Legal Name"
	newTradeName := "New Trade Name"
	newTaxID := "New Tax ID"
	newEmail := "new@example.com"
	newPhone := "+1 555 9999"
	newWebsite := "https://new.example"
	newAddress := Address{Street: "New St", City: "New City", Country: "New Country"}

	patch := LegalEntityPatch{
		Type:           &newType,
		LegalName:      &newLegalName,
		TradeName:      &newTradeName,
		TaxID:          &newTaxID,
		Email:          &newEmail,
		Phone:          &newPhone,
		Website:        &newWebsite,
		BillingAddress: &newAddress,
	}

	entity := original
	beforeApply := entity.UpdatedAt
	entity.ApplyPatch(patch)

	// All fields should be replaced
	if entity.Type != newType {
		t.Fatalf("Type = %q, want %q", entity.Type, newType)
	}
	if entity.LegalName != newLegalName {
		t.Fatalf("LegalName = %q, want %q", entity.LegalName, newLegalName)
	}
	if entity.TradeName != newTradeName {
		t.Fatalf("TradeName = %q, want %q", entity.TradeName, newTradeName)
	}
	if entity.TaxID != newTaxID {
		t.Fatalf("TaxID = %q, want %q", entity.TaxID, newTaxID)
	}
	if entity.Email != newEmail {
		t.Fatalf("Email = %q, want %q", entity.Email, newEmail)
	}
	if entity.Phone != newPhone {
		t.Fatalf("Phone = %q, want %q", entity.Phone, newPhone)
	}
	if entity.Website != newWebsite {
		t.Fatalf("Website = %q, want %q", entity.Website, newWebsite)
	}
	if entity.BillingAddress != newAddress {
		t.Fatalf("BillingAddress = %#v, want %#v", entity.BillingAddress, newAddress)
	}

	// Unchangeable fields should remain unchanged
	if entity.ID != original.ID {
		t.Fatalf("ID = %q, want %q (ID should not change)", entity.ID, original.ID)
	}
	if entity.CreatedAt != original.CreatedAt {
		t.Fatal("CreatedAt should not change")
	}

	// UpdatedAt should be updated
	if entity.UpdatedAt.Before(beforeApply) || entity.UpdatedAt.Equal(beforeApply) {
		t.Fatal("UpdatedAt should be more recent after patching")
	}
}

func TestLegalEntity_ValidateDelete(t *testing.T) {
	t.Parallel()

	// ValidateDelete is a seam for future relationship-protection logic.
	entity := LegalEntity{
		ID:        "le_test123",
		Type:      EntityTypeCompany,
		LegalName: "Test Entity",
	}

	err := entity.ValidateDelete()
	if err != nil {
		t.Fatalf("ValidateDelete() = %v, want nil", err)
	}
}
