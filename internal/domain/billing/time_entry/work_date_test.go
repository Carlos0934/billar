package timeentry_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/domain/billing/time_entry"
)

func TestNewWorkDateNormalizesToDateOnlyUTC(t *testing.T) {
	workDate, err := timeentry.NewWorkDate(time.Date(2026, time.March, 30, 14, 45, 59, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewWorkDate() error = %v", err)
	}

	want := time.Date(2026, time.March, 30, 0, 0, 0, 0, time.UTC)
	if got := workDate.Time(); !got.Equal(want) {
		t.Fatalf("Time() = %v, want %v", got, want)
	}
}

func TestNewWorkDateRejectsZeroDate(t *testing.T) {
	_, err := timeentry.NewWorkDate(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero work date")
	}
	if !errors.Is(err, timeentry.ErrWorkDateRequired) {
		t.Fatalf("err = %v, want %v", err, timeentry.ErrWorkDateRequired)
	}
}
