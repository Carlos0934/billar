package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrServiceAgreementNotFound = errors.New("service agreement not found")

// ServiceAgreementStore is the persistence port for ServiceAgreement entities.
type ServiceAgreementStore interface {
	Save(ctx context.Context, agreement *core.ServiceAgreement) error
	GetByID(ctx context.Context, id string) (*core.ServiceAgreement, error)
	ListByCustomerProfileID(ctx context.Context, customerProfileID string) ([]core.ServiceAgreement, error)
}

// AgreementService orchestrates creation, rate updates, and lifecycle management
// of ServiceAgreement entities.
type AgreementService struct {
	agreements ServiceAgreementStore
	profiles   CustomerProfileStore
}

// NewAgreementService constructs an AgreementService with injected stores.
func NewAgreementService(agreements ServiceAgreementStore, profiles CustomerProfileStore) AgreementService {
	return AgreementService{agreements: agreements, profiles: profiles}
}

// Create validates the customer profile exists, constructs a new agreement, and persists it.
func (s AgreementService) Create(ctx context.Context, cmd CreateServiceAgreementCommand) (ServiceAgreementDTO, error) {
	if _, err := s.getCustomerProfile(ctx, cmd.CustomerProfileID); err != nil {
		return ServiceAgreementDTO{}, err
	}

	sa, err := core.NewServiceAgreement(core.ServiceAgreementParams{
		CustomerProfileID: cmd.CustomerProfileID,
		Name:              cmd.Name,
		Description:       cmd.Description,
		BillingMode:       core.BillingMode(cmd.BillingMode),
		HourlyRate:        cmd.HourlyRate,
		Currency:          cmd.Currency,
		ValidFrom:         cmd.ValidFrom,
		ValidUntil:        cmd.ValidUntil,
	})
	if err != nil {
		return ServiceAgreementDTO{}, err
	}

	if err := s.agreements.Save(ctx, &sa); err != nil {
		return ServiceAgreementDTO{}, fmt.Errorf("save service agreement: %w", err)
	}

	return serviceAgreementToDTO(sa), nil
}

// Get retrieves a service agreement by ID and returns its canonical DTO.
func (s AgreementService) Get(ctx context.Context, id string) (ServiceAgreementDTO, error) {
	sa, err := s.getServiceAgreement(ctx, id)
	if err != nil {
		return ServiceAgreementDTO{}, err
	}
	return serviceAgreementToDTO(*sa), nil
}

// UpdateRate fetches, mutates the rate, persists, and returns the updated DTO.
func (s AgreementService) UpdateRate(ctx context.Context, id string, cmd UpdateServiceAgreementRateCommand) (ServiceAgreementDTO, error) {
	sa, err := s.getServiceAgreement(ctx, id)
	if err != nil {
		return ServiceAgreementDTO{}, err
	}

	if err := sa.UpdateRate(cmd.HourlyRate); err != nil {
		return ServiceAgreementDTO{}, err
	}

	if err := s.agreements.Save(ctx, sa); err != nil {
		return ServiceAgreementDTO{}, fmt.Errorf("save service agreement: %w", err)
	}

	return serviceAgreementToDTO(*sa), nil
}

// Activate enables the agreement and persists the change.
func (s AgreementService) Activate(ctx context.Context, id string) (ServiceAgreementDTO, error) {
	sa, err := s.getServiceAgreement(ctx, id)
	if err != nil {
		return ServiceAgreementDTO{}, err
	}

	sa.Activate()

	if err := s.agreements.Save(ctx, sa); err != nil {
		return ServiceAgreementDTO{}, fmt.Errorf("save service agreement: %w", err)
	}

	return serviceAgreementToDTO(*sa), nil
}

// Deactivate disables the agreement and persists the change.
func (s AgreementService) Deactivate(ctx context.Context, id string) (ServiceAgreementDTO, error) {
	sa, err := s.getServiceAgreement(ctx, id)
	if err != nil {
		return ServiceAgreementDTO{}, err
	}

	sa.Deactivate()

	if err := s.agreements.Save(ctx, sa); err != nil {
		return ServiceAgreementDTO{}, fmt.Errorf("save service agreement: %w", err)
	}

	return serviceAgreementToDTO(*sa), nil
}

// ListByCustomerProfile delegates to the store and maps results to DTOs.
func (s AgreementService) ListByCustomerProfile(ctx context.Context, profileID string) ([]ServiceAgreementDTO, error) {
	agreements, err := s.agreements.ListByCustomerProfileID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("list service agreements: %w", err)
	}

	dtos := make([]ServiceAgreementDTO, 0, len(agreements))
	for _, sa := range agreements {
		dtos = append(dtos, serviceAgreementToDTO(sa))
	}
	return dtos, nil
}

func (s AgreementService) getServiceAgreement(ctx context.Context, id string) (*core.ServiceAgreement, error) {
	sa, err := s.agreements.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrServiceAgreementNotFound) {
			return nil, ErrServiceAgreementNotFound
		}
		return nil, fmt.Errorf("get service agreement: %w", err)
	}
	return sa, nil
}

func (s AgreementService) getCustomerProfile(ctx context.Context, id string) (*core.CustomerProfile, error) {
	profile, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return nil, fmt.Errorf("create service agreement: customer profile not found: %w", err)
		}
		return nil, fmt.Errorf("create service agreement: get customer profile: %w", err)
	}
	return profile, nil
}
