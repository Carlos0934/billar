package billingvalues

import (
	"time"
)

type DateRange struct {
	start time.Time
	end   time.Time
}

func NewDateRange(start, end time.Time) (DateRange, error) {
	normalizedStart := normalizeDate(start)
	normalizedEnd := normalizeDate(end)
	if normalizedStart.IsZero() {
		return DateRange{}, ErrStartDateRequired
	}
	if normalizedEnd.IsZero() {
		return DateRange{}, ErrEndDateRequired
	}
	if normalizedEnd.Before(normalizedStart) {
		return DateRange{}, ErrEndDateBeforeStartDate
	}

	return DateRange{start: normalizedStart, end: normalizedEnd}, nil
}

func (dateRange DateRange) Start() time.Time {
	return dateRange.start
}

func (dateRange DateRange) End() time.Time {
	return dateRange.end
}

func (dateRange DateRange) Contains(date time.Time) bool {
	normalizedDate := normalizeDate(date)
	return !normalizedDate.Before(dateRange.start) && !normalizedDate.After(dateRange.end)
}

func normalizeDate(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
