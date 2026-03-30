package billingvalues

import "fmt"

const hoursScale = int64(10_000)

type Hours struct {
	value int64
}

func NewHours(value int64) (Hours, error) {
	if value < 0 {
		return Hours{}, ErrHoursNegative
	}

	return Hours{value: value}, nil
}

func (hours Hours) Value() int64 {
	return hours.value
}

func (hours Hours) IsZero() bool {
	return hours.value == 0
}

func (hours Hours) String() string {
	whole := hours.value / hoursScale
	frac := hours.value % hoursScale
	return fmt.Sprintf("%d.%04d", whole, frac)
}
