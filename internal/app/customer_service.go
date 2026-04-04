package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrCustomerListAccessDenied = errors.New("customer list requires an authenticated identity")

var ErrCustomerCreateAccessDenied = errors.New("customer create requires an authenticated identity")
var ErrCustomerUpdateAccessDenied = errors.New("customer update requires an authenticated identity")
var ErrCustomerDeleteAccessDenied = errors.New("customer delete requires an authenticated identity")
var ErrCustomerNotFound = errors.New("customer not found")

type CustomerStore interface {
	List(ctx context.Context, query ListQuery) (ListResult[core.Customer], error)
	Save(ctx context.Context, customer *core.Customer) error
	GetByID(ctx context.Context, id string) (*core.Customer, error)
	Delete(ctx context.Context, id string) error
}

type CustomerService struct {
	identities AuthenticatedIdentitySource
	store      CustomerStore
}

func NewCustomerService(identities AuthenticatedIdentitySource, store CustomerStore) CustomerService {
	return CustomerService{identities: identities, store: store}
}

func (s CustomerService) List(ctx context.Context, query ListQuery) (ListResult[CustomerDTO], error) {
	query = query.Normalize()

	if s.identities == nil {
		return ListResult[CustomerDTO]{}, errors.New("customer authenticated identity source is required")
	}
	if s.store == nil {
		return ListResult[CustomerDTO]{}, errors.New("customer store is required")
	}

	_, ok, err := s.identities.CurrentIdentity(ctx)
	if err != nil {
		return ListResult[CustomerDTO]{}, fmt.Errorf("load authenticated identity: %w", err)
	}
	if !ok {
		return ListResult[CustomerDTO]{}, ErrCustomerListAccessDenied
	}

	result, err := s.store.List(ctx, query)
	if err != nil {
		return ListResult[CustomerDTO]{}, fmt.Errorf("list customers: %w", err)
	}

	items := make([]CustomerDTO, 0, len(result.Items))
	for _, customer := range result.Items {
		items = append(items, customerToDTO(customer))
	}

	return ListResult[CustomerDTO]{
		Items:    items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

func (s CustomerService) Create(ctx context.Context, cmd CreateCustomerCommand) (CustomerDTO, error) {
	if s.identities == nil {
		return CustomerDTO{}, errors.New("customer authenticated identity source is required")
	}
	if s.store == nil {
		return CustomerDTO{}, errors.New("customer store is required")
	}

	_, ok, err := s.identities.CurrentIdentity(ctx)
	if err != nil {
		return CustomerDTO{}, fmt.Errorf("load authenticated identity: %w", err)
	}
	if !ok {
		return CustomerDTO{}, ErrCustomerCreateAccessDenied
	}

	customerType := core.CustomerType(cmd.Type)
	if !customerType.IsValid() {
		return CustomerDTO{}, fmt.Errorf("invalid customer type: %s", cmd.Type)
	}

	customer, err := core.NewCustomer(core.CustomerParams{
		Type:           customerType,
		LegalName:      cmd.LegalName,
		TradeName:      cmd.TradeName,
		TaxID:          cmd.TaxID,
		Email:          cmd.Email,
		Phone:          cmd.Phone,
		Website:        cmd.Website,
		BillingAddress: addressFromDTO(cmd.BillingAddress),
		Notes:          cmd.Notes,
	})
	if err != nil {
		return CustomerDTO{}, err
	}

	if err := s.store.Save(ctx, &customer); err != nil {
		return CustomerDTO{}, fmt.Errorf("save customer: %w", err)
	}

	return customerToDTO(customer), nil
}

func addressFromDTO(dto AddressDTO) core.Address {
	return core.Address{
		Street:     dto.Street,
		City:       dto.City,
		State:      dto.State,
		PostalCode: dto.PostalCode,
		Country:    dto.Country,
	}
}

func patchToCorePatch(cmd PatchCustomerCommand) core.CustomerPatch {
	patch := core.CustomerPatch{}
	if cmd.Type != nil {
		t := core.CustomerType(*cmd.Type)
		patch.Type = &t
	}
	if cmd.LegalName != nil {
		patch.LegalName = cmd.LegalName
	}
	if cmd.TradeName != nil {
		patch.TradeName = cmd.TradeName
	}
	if cmd.TaxID != nil {
		patch.TaxID = cmd.TaxID
	}
	if cmd.Email != nil {
		patch.Email = cmd.Email
	}
	if cmd.Phone != nil {
		patch.Phone = cmd.Phone
	}
	if cmd.Website != nil {
		patch.Website = cmd.Website
	}
	if cmd.BillingAddress != nil {
		addr := addressFromDTO(*cmd.BillingAddress)
		patch.BillingAddress = &addr
	}
	if cmd.Notes != nil {
		patch.Notes = cmd.Notes
	}
	if cmd.DefaultCurrency != nil {
		patch.DefaultCurrency = cmd.DefaultCurrency
	}
	return patch
}

func (s CustomerService) Update(ctx context.Context, id string, cmd PatchCustomerCommand) (CustomerDTO, error) {
	if s.identities == nil {
		return CustomerDTO{}, errors.New("customer authenticated identity source is required")
	}
	if s.store == nil {
		return CustomerDTO{}, errors.New("customer store is required")
	}

	_, ok, err := s.identities.CurrentIdentity(ctx)
	if err != nil {
		return CustomerDTO{}, fmt.Errorf("load authenticated identity: %w", err)
	}
	if !ok {
		return CustomerDTO{}, ErrCustomerUpdateAccessDenied
	}

	customer, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return CustomerDTO{}, ErrCustomerNotFound
		}
		return CustomerDTO{}, fmt.Errorf("get customer: %w", err)
	}

	patch := patchToCorePatch(cmd)
	customer.ApplyPatch(patch)

	// Re-validate the resulting entity after applying the patch
	if err := customer.Validate(); err != nil {
		return CustomerDTO{}, fmt.Errorf("validate customer: %w", err)
	}

	if err := s.store.Save(ctx, customer); err != nil {
		return CustomerDTO{}, fmt.Errorf("save customer: %w", err)
	}

	return customerToDTO(*customer), nil
}

func (s CustomerService) Delete(ctx context.Context, id string) error {
	if s.identities == nil {
		return errors.New("customer authenticated identity source is required")
	}
	if s.store == nil {
		return errors.New("customer store is required")
	}

	_, ok, err := s.identities.CurrentIdentity(ctx)
	if err != nil {
		return fmt.Errorf("load authenticated identity: %w", err)
	}
	if !ok {
		return ErrCustomerDeleteAccessDenied
	}

	customer, err := s.store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return ErrCustomerNotFound
		}
		return fmt.Errorf("get customer: %w", err)
	}

	if err := customer.ValidateDelete(); err != nil {
		return err
	}

	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete customer: %w", err)
	}

	return nil
}
