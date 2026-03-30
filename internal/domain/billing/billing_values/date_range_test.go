package billingvalues_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/billing_values"
)

func TestNewDateRange(t *testing.T) {
	start := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)

	dateRange, err := billingvalues.NewDateRange(start, end)
	if err != nil {
		t.Fatalf("NewDateRange() error = %v", err)
	}

	if got := dateRange.Start(); !got.Equal(start) {
		t.Fatalf("start = %v, want %v", got, start)
	}

	if got := dateRange.End(); !got.Equal(end) {
		t.Fatalf("end = %v, want %v", got, end)
	}

	if !dateRange.Contains(time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected mid-range date to be contained")
	}

	if dateRange.Contains(time.Date(2026, time.January, 25, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expected out-of-range date to be excluded")
	}
}

func TestNewDateRangeRejectsEndBeforeStart(t *testing.T) {
	start := time.Date(2026, time.January, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC)

	if _, err := billingvalues.NewDateRange(start, end); err == nil {
		t.Fatal("expected error for invalid date range")
	} else if !errors.Is(err, billingvalues.ErrEndDateBeforeStartDate) {
		t.Fatalf("err = %v, want %v", err, billingvalues.ErrEndDateBeforeStartDate)
	}
}
