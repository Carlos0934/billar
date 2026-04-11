package core

import "errors"

// ErrHoursNotPositive is returned when a non-positive Hours value is provided.
var ErrHoursNotPositive = errors.New("hours must be positive")

// Hours represents a time duration in 4-decimal integer precision.
// A value of 15000 means 1.5000 hours. Only strictly positive values are valid.
type Hours int64

// NewHours constructs a valid Hours value. The value must be strictly positive.
func NewHours(val int64) (Hours, error) {
	if val <= 0 {
		return 0, ErrHoursNotPositive
	}
	return Hours(val), nil
}

// IsPositive reports whether the hours value is strictly positive.
func (h Hours) IsPositive() bool {
	return h > 0
}
