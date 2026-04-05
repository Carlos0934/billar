package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrCustomerProfileNotFound = errors.New("customer profile not found")

type CustomerProfileStore interface {
	List(ctx context.Context, query ListQuery) (ListResult[core.CustomerProfile], error)
	Save(ctx context.Context, profile *core.CustomerProfile) error
	GetByID(ctx context.Context, id string) (*core.CustomerProfile, error)
	Delete(ctx context.Context, id string) error
}

type CustomerProfileService struct {
	legalEntities LegalEntityStore
	profiles      CustomerProfileStore
}

func NewCustomerProfileService(legalEntities LegalEntityStore, profiles CustomerProfileStore) CustomerProfileService {
	return CustomerProfileService{legalEntities: legalEntities, profiles: profiles}
}

func (s CustomerProfileService) List(ctx context.Context, query ListQuery) (ListResult[CustomerProfileDTO], error) {
	query = query.Normalize()

	if s.profiles == nil {
		return ListResult[CustomerProfileDTO]{}, errors.New("customer profile store is required")
	}

	result, err := s.profiles.List(ctx, query)
	if err != nil {
		return ListResult[CustomerProfileDTO]{}, fmt.Errorf("list customer profiles: %w", err)
	}

	items := make([]CustomerProfileDTO, 0, len(result.Items))
	for _, profile := range result.Items {
		items = append(items, customerProfileToDTO(profile))
	}

	return ListResult[CustomerProfileDTO]{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s CustomerProfileService) Create(ctx context.Context, cmd CreateCustomerProfileCommand) (CustomerProfileDTO, error) {
	if s.legalEntities == nil {
		return CustomerProfileDTO{}, errors.New("legal entity store is required")
	}
	if s.profiles == nil {
		return CustomerProfileDTO{}, errors.New("customer profile store is required")
	}

	// Create the legal entity inline.
	leSvc := LegalEntityService{store: s.legalEntities}
	leDTO, err := leSvc.Create(ctx, CreateLegalEntityCommand{
		Type:           cmd.LegalEntityType,
		LegalName:      cmd.LegalName,
		TradeName:      cmd.TradeName,
		TaxID:          cmd.TaxID,
		Email:          cmd.Email,
		Phone:          cmd.Phone,
		Website:        cmd.Website,
		BillingAddress: cmd.BillingAddress,
	})
	if err != nil {
		return CustomerProfileDTO{}, fmt.Errorf("create legal entity: %w", err)
	}

	profile, err := core.NewCustomerProfile(core.CustomerProfileParams{
		LegalEntityID:   leDTO.ID,
		DefaultCurrency: cmd.DefaultCurrency,
		Notes:           cmd.Notes,
	})
	if err != nil {
		return CustomerProfileDTO{}, err
	}

	if err := s.profiles.Save(ctx, &profile); err != nil {
		return CustomerProfileDTO{}, fmt.Errorf("save customer profile: %w", err)
	}

	return customerProfileToDTO(profile), nil
}

func (s CustomerProfileService) Get(ctx context.Context, id string) (CustomerProfileDTO, error) {
	if s.profiles == nil {
		return CustomerProfileDTO{}, errors.New("customer profile store is required")
	}

	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return CustomerProfileDTO{}, ErrCustomerProfileNotFound
		}
		return CustomerProfileDTO{}, fmt.Errorf("get customer profile: %w", err)
	}

	return customerProfileToDTO(*profile), nil
}

func (s CustomerProfileService) Update(ctx context.Context, id string, cmd PatchCustomerProfileCommand) (CustomerProfileDTO, error) {
	if s.profiles == nil {
		return CustomerProfileDTO{}, errors.New("customer profile store is required")
	}

	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return CustomerProfileDTO{}, ErrCustomerProfileNotFound
		}
		return CustomerProfileDTO{}, fmt.Errorf("get customer profile: %w", err)
	}

	// Cascade legal entity fields when present.
	if cmd.hasLegalEntityFields() {
		if s.legalEntities == nil {
			return CustomerProfileDTO{}, errors.New("legal entity store is required")
		}
		leSvc := LegalEntityService{store: s.legalEntities}
		if _, err := leSvc.Update(ctx, profile.LegalEntityID, cmd.toLegalEntityPatch()); err != nil {
			return CustomerProfileDTO{}, fmt.Errorf("update legal entity: %w", err)
		}
	}

	patch := patchToCoreCustomerProfilePatch(cmd)
	profile.ApplyPatch(patch)

	// Re-validate the resulting profile after applying the patch
	if err := profile.Validate(); err != nil {
		return CustomerProfileDTO{}, fmt.Errorf("validate customer profile: %w", err)
	}

	if err := s.profiles.Save(ctx, profile); err != nil {
		return CustomerProfileDTO{}, fmt.Errorf("save customer profile: %w", err)
	}

	return customerProfileToDTO(*profile), nil
}

func (s CustomerProfileService) Delete(ctx context.Context, id string) error {
	if s.profiles == nil {
		return errors.New("customer profile store is required")
	}

	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return ErrCustomerProfileNotFound
		}
		return fmt.Errorf("get customer profile: %w", err)
	}

	if err := profile.ValidateDelete(); err != nil {
		return err
	}

	if err := s.profiles.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete customer profile: %w", err)
	}

	return nil
}
