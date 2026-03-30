package customer

type CustomerType string

const (
	TypeIndividual CustomerType = "individual"
	TypeCompany    CustomerType = "company"
)

func (customerType CustomerType) validate() error {
	switch customerType {
	case TypeIndividual, TypeCompany:
		return nil
	default:
		return ErrCustomerTypeInvalid
	}
}

type CustomerStatus string

const (
	StatusActive   CustomerStatus = "active"
	StatusInactive CustomerStatus = "inactive"
)
