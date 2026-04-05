package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrIssuerProfileNotFound = errors.New("issuer profile not found")

type IssuerProfileStore interface {
	Save(ctx context.Context, profile *core.IssuerProfile) error
	GetByID(ctx context.Context, id string) (*core.IssuerProfile, error)
	Delete(ctx context.Context, id string) error
}

type IssuerProfileService struct {
	legalEntities LegalEntityStore
	profiles      IssuerProfileStore
}

func NewIssuerProfileService(legalEntities LegalEntityStore, profiles IssuerProfileStore) IssuerProfileService {
	return IssuerProfileService{legalEntities: legalEntities, profiles: profiles}
}

func (s IssuerProfileService) Create(ctx context.Context, cmd CreateIssuerProfileCommand) (IssuerProfileDTO, error) {
	if s.legalEntities == nil {
		return IssuerProfileDTO{}, errors.New("legal entity store is required")
	}
	if s.profiles == nil {
		return IssuerProfileDTO{}, errors.New("issuer profile store is required")
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
		return IssuerProfileDTO{}, fmt.Errorf("create legal entity: %w", err)
	}

	profile, err := core.NewIssuerProfile(core.IssuerProfileParams{
		LegalEntityID:   leDTO.ID,
		DefaultCurrency: cmd.DefaultCurrency,
		DefaultNotes:    cmd.DefaultNotes,
	})
	if err != nil {
		return IssuerProfileDTO{}, err
	}

	if err := s.profiles.Save(ctx, &profile); err != nil {
		return IssuerProfileDTO{}, fmt.Errorf("save issuer profile: %w", err)
	}

	return issuerProfileToDTO(profile), nil
}

func (s IssuerProfileService) Get(ctx context.Context, id string) (IssuerProfileDTO, error) {
	if s.profiles == nil {
		return IssuerProfileDTO{}, errors.New("issuer profile store is required")
	}

	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrIssuerProfileNotFound) {
			return IssuerProfileDTO{}, ErrIssuerProfileNotFound
		}
		return IssuerProfileDTO{}, fmt.Errorf("get issuer profile: %w", err)
	}

	return issuerProfileToDTO(*profile), nil
}

func (s IssuerProfileService) Update(ctx context.Context, id string, cmd PatchIssuerProfileCommand) (IssuerProfileDTO, error) {
	if s.profiles == nil {
		return IssuerProfileDTO{}, errors.New("issuer profile store is required")
	}

	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrIssuerProfileNotFound) {
			return IssuerProfileDTO{}, ErrIssuerProfileNotFound
		}
		return IssuerProfileDTO{}, fmt.Errorf("get issuer profile: %w", err)
	}

	// Cascade legal entity fields when present.
	if cmd.hasLegalEntityFields() {
		if s.legalEntities == nil {
			return IssuerProfileDTO{}, errors.New("legal entity store is required")
		}
		leSvc := LegalEntityService{store: s.legalEntities}
		if _, err := leSvc.Update(ctx, profile.LegalEntityID, cmd.toLegalEntityPatch()); err != nil {
			return IssuerProfileDTO{}, fmt.Errorf("update legal entity: %w", err)
		}
	}

	patch := patchToCoreIssuerProfilePatch(cmd)
	profile.ApplyPatch(patch)

	// Re-validate the resulting profile after applying the patch
	if err := profile.Validate(); err != nil {
		return IssuerProfileDTO{}, fmt.Errorf("validate issuer profile: %w", err)
	}

	if err := s.profiles.Save(ctx, profile); err != nil {
		return IssuerProfileDTO{}, fmt.Errorf("save issuer profile: %w", err)
	}

	return issuerProfileToDTO(*profile), nil
}

func (s IssuerProfileService) Delete(ctx context.Context, id string) error {
	if s.profiles == nil {
		return errors.New("issuer profile store is required")
	}

	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrIssuerProfileNotFound) {
			return ErrIssuerProfileNotFound
		}
		return fmt.Errorf("get issuer profile: %w", err)
	}

	if err := profile.ValidateDelete(); err != nil {
		return err
	}

	if err := s.profiles.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete issuer profile: %w", err)
	}

	return nil
}
