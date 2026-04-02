package app

import (
	"context"
	"testing"
)

func TestHealthServiceStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		appName string
		want    HealthDTO
	}{
		{
			name:    "uses configured app name",
			appName: "billar-cli",
			want: HealthDTO{
				Name:   "billar-cli",
				Status: "ok",
			},
		},
		{
			name:    "falls back to default app name",
			appName: "   ",
			want: HealthDTO{
				Name:   "billar",
				Status: "ok",
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewHealthService(tc.appName)

			got, err := svc.Status(context.Background())
			if err != nil {
				t.Fatalf("Status() error = %v", err)
			}

			if got != tc.want {
				t.Fatalf("Status() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
