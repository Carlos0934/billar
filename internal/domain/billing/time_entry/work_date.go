package timeentry

import "time"

type WorkDate struct {
	value time.Time
}

func NewWorkDate(value time.Time) (WorkDate, error) {
	if value.IsZero() {
		return WorkDate{}, ErrWorkDateRequired
	}

	return WorkDate{
		value: time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC),
	}, nil
}

func (date WorkDate) Time() time.Time {
	return date.value
}

func (date WorkDate) IsZero() bool {
	return date.value.IsZero()
}
