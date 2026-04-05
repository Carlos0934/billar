package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrLegalEntityNotFound = errors.New("legal entity not found")

type LegalEntityStore interface {
	List(ctx context.Context, query ListQuery) (ListResult[core.LegalEntity], error)
	Save(ctx context.Context, entity *core.LegalEntity) error
	GetByID(ctx context.Context, id string) (*core.LegalEntity, error)
	Delete(ctx context.Context, id string) error
}

type LegalEntityService struct {
	store LegalEntityStore
}

func NewLegalEntityService(store LegalEntityStore) LegalEntityService {
	return LegalEntityService{store: store}
}

func (s LegalEntityService) List(ctx context.Context, query ListQuery) (ListResult[LegalEntityDTO], error) {
	query = query.Normalize()

	if s.store == nil {
		return ListResult[LegalEntityDTO]{}, errors.New("legal entity store is required")
	}

	result, err := s.store.List(ctx, query)
	if err != nil {
		return ListResult[LegalEntityDTO]{}, fmt.Errorf("list legal entities: %w", err)
	}

	items := make([]LegalEntityDTO, 0, len(result.Items))
	for _, entity := range result.Items {
		items = append(items, legalEntityToDTO(entity))
	}

	return ListResult[LegalEntityDTO]{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s LegalEntityService) Create(ctx context.Context, cmd CreateLegalEntityCommand) (LegalEntityDTO, error) {
	if s.store == nil {
		return LegalEntityDTO{}, errors.New("legal entity store is required")
	}

	entityType := core.EntityType(cmd.Type)
	if !entityType.IsValid() {
		return LegalEntityDTO{}, fmt.Errorf("invalid entity type: %s", cmd.Type)
	}

	entity, err := core.NewLegalEntity(core.LegalEntityParams{
		Type:           entityType,
		LegalName:      cmd.LegalName,
		TradeName:      cmd.TradeName,
		TaxID:          cmd.TaxID,
		Email:          cmd.Email,
		Phone:          cmd.Phone,
		Website:        cmd.Website,
		BillingAddress: addressFromDTO(cmd.BillingAddress),
	})
	if err != nil {
		return LegalEntityDTO{}, err
	}

	if err := s.store.Save(ctx, &entity); err != nil {
		return LegalEntityDTO{}, fmt.Errorf("save legal entity: %w", err)
	}

	return legalEntityToDTO(entity), nil
}

func (s LegalEntityService) Get(ctx context.Context, id string) (LegalEntityDTO, error) {
	if s.store == nil {
		return LegalEntityDTO{}, errors.New("legal entity store is required")
	}

	entity, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrLegalEntityNotFound) {
			return LegalEntityDTO{}, ErrLegalEntityNotFound
		}
		return LegalEntityDTO{}, fmt.Errorf("get legal entity: %w", err)
	}

	return legalEntityToDTO(*entity), nil
}

func (s LegalEntityService) Update(ctx context.Context, id string, cmd PatchLegalEntityCommand) (LegalEntityDTO, error) {
	if s.store == nil {
		return LegalEntityDTO{}, errors.New("legal entity store is required")
	}

	entity, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrLegalEntityNotFound) {
			return LegalEntityDTO{}, ErrLegalEntityNotFound
		}
		return LegalEntityDTO{}, fmt.Errorf("get legal entity: %w", err)
	}

	patch := patchToCoreLegalEntityPatch(cmd)
	entity.ApplyPatch(patch)

	// Re-validate the resulting entity after applying the patch
	if err := entity.Validate(); err != nil {
		return LegalEntityDTO{}, fmt.Errorf("validate legal entity: %w", err)
	}

	if err := s.store.Save(ctx, entity); err != nil {
		return LegalEntityDTO{}, fmt.Errorf("save legal entity: %w", err)
	}

	return legalEntityToDTO(*entity), nil
}

func (s LegalEntityService) Delete(ctx context.Context, id string) error {
	if s.store == nil {
		return errors.New("legal entity store is required")
	}

	entity, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrLegalEntityNotFound) {
			return ErrLegalEntityNotFound
		}
		return fmt.Errorf("get legal entity: %w", err)
	}

	if err := entity.ValidateDelete(); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete legal entity: %w", err)
	}

	return nil
}
