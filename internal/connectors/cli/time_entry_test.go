package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Carlos0934/billar/internal/app"
)

// stubTimeEntryService is the test double for TimeEntryServiceProvider.
type stubTimeEntryService struct {
	recordArg       *app.RecordTimeEntryCommand
	recordRes       app.TimeEntryDTO
	recordErr       error
	getID           string
	getRes          app.TimeEntryDTO
	getErr          error
	updateArg       *app.UpdateTimeEntryCommand
	updateRes       app.TimeEntryDTO
	updateErr       error
	deleteID        string
	deleteErr       error
	listProfileID   string
	listRes         []app.TimeEntryDTO
	listErr         error
	listUnbilledID  string
	listUnbilledRes []app.TimeEntryDTO
	listUnbilledErr error
}

func (s *stubTimeEntryService) Record(ctx context.Context, cmd app.RecordTimeEntryCommand) (app.TimeEntryDTO, error) {
	_ = ctx
	s.recordArg = &cmd
	return s.recordRes, s.recordErr
}

func (s *stubTimeEntryService) Get(ctx context.Context, id string) (app.TimeEntryDTO, error) {
	_ = ctx
	s.getID = id
	return s.getRes, s.getErr
}

func (s *stubTimeEntryService) UpdateEntry(ctx context.Context, cmd app.UpdateTimeEntryCommand) (app.TimeEntryDTO, error) {
	_ = ctx
	s.updateArg = &cmd
	return s.updateRes, s.updateErr
}

func (s *stubTimeEntryService) DeleteEntry(ctx context.Context, id string) error {
	_ = ctx
	s.deleteID = id
	return s.deleteErr
}

func (s *stubTimeEntryService) ListByCustomerProfile(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error) {
	_ = ctx
	s.listProfileID = customerID
	return s.listRes, s.listErr
}

func (s *stubTimeEntryService) ListUnbilled(ctx context.Context, customerID string) ([]app.TimeEntryDTO, error) {
	_ = ctx
	s.listUnbilledID = customerID
	return s.listUnbilledRes, s.listUnbilledErr
}

// newTestTimeEntryCommand builds a minimal Command with health stub and the given time entry service.
func newTestTimeEntryCommand(svc TimeEntryServiceProvider) Command {
	return NewCommand(stubHealthService{status: app.HealthDTO{Name: "billar", Status: "ok"}}, nil, nil, nil, nil, svc, false)
}

// -- time-entry record --

func TestTimeEntryRecordCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubTimeEntryService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
	}{
		{
			name: "records time entry from json",
			args: []string{"time-entry", "record", `--json={"customer_profile_id":"cus_1","service_agreement_id":"sa_1","description":"Write tests","hours":10000,"billable":true,"date":"2026-04-10T00:00:00Z"}`},
			svc: &stubTimeEntryService{
				recordRes: app.TimeEntryDTO{
					ID:                 "te_abc",
					CustomerProfileID:  "cus_1",
					ServiceAgreementID: "sa_1",
					Description:        "Write tests",
					Hours:              10000,
					Billable:           true,
				},
			},
			wantContain: "te_abc",
		},
		{
			name:          "returns error when --json is missing",
			args:          []string{"time-entry", "record"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--json",
		},
		{
			name:          "returns error when json is invalid",
			args:          []string{"time-entry", "record", "--json=not-json"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "invalid",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestTimeEntryCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- time-entry get --

func TestTimeEntryGetCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubTimeEntryService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
		wantGetID     string
	}{
		{
			name: "gets time entry by id",
			args: []string{"time-entry", "get", "--id=te_abc"},
			svc: &stubTimeEntryService{
				getRes: app.TimeEntryDTO{
					ID:          "te_abc",
					Description: "Write tests",
					Hours:       10000,
				},
			},
			wantGetID:   "te_abc",
			wantContain: "te_abc",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"time-entry", "get"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
		{
			name: "propagates service not found error",
			args: []string{"time-entry", "get", "--id=te_nonexistent"},
			svc: &stubTimeEntryService{
				getErr: errors.New("time entry not found"),
			},
			wantErr:       true,
			wantErrSubstr: "not found",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestTimeEntryCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantGetID != "" && tc.svc.getID != tc.wantGetID {
				t.Errorf("Get() id = %q, want %q", tc.svc.getID, tc.wantGetID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- time-entry update --

func TestTimeEntryUpdateCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubTimeEntryService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
	}{
		{
			name: "updates time entry from json",
			args: []string{"time-entry", "update", "--id=te_abc", `--json={"description":"Updated","hours":20000}`},
			svc: &stubTimeEntryService{
				updateRes: app.TimeEntryDTO{
					ID:          "te_abc",
					Description: "Updated",
					Hours:       20000,
				},
			},
			wantContain: "te_abc",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"time-entry", "update", `--json={"description":"x","hours":1}`},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
		{
			name:          "returns error when --json is missing",
			args:          []string{"time-entry", "update", "--id=te_abc"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--json",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestTimeEntryCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- time-entry delete --

func TestTimeEntryDeleteCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubTimeEntryService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
		wantDeleteID  string
	}{
		{
			name:         "deletes time entry",
			args:         []string{"time-entry", "delete", "--id=te_abc"},
			svc:          &stubTimeEntryService{},
			wantDeleteID: "te_abc",
			wantContain:  "te_abc",
		},
		{
			name:          "returns error when --id is missing",
			args:          []string{"time-entry", "delete"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--id",
		},
		{
			name: "propagates service error",
			args: []string{"time-entry", "delete", "--id=te_locked"},
			svc: &stubTimeEntryService{
				deleteErr: errors.New("time entry is locked"),
			},
			wantErr:       true,
			wantErrSubstr: "locked",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestTimeEntryCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantDeleteID != "" && tc.svc.deleteID != tc.wantDeleteID {
				t.Errorf("DeleteEntry() id = %q, want %q", tc.svc.deleteID, tc.wantDeleteID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- time-entry list --

func TestTimeEntryListCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		svc           *stubTimeEntryService
		wantErr       bool
		wantErrSubstr string
		wantContain   string
		wantProfileID string
	}{
		{
			name: "lists time entries for customer profile",
			args: []string{"time-entry", "list", "--customer-id=cus_1"},
			svc: &stubTimeEntryService{
				listRes: []app.TimeEntryDTO{
					{ID: "te_001", Description: "Task A", Hours: 5000},
				},
			},
			wantProfileID: "cus_1",
			wantContain:   "te_001",
		},
		{
			name:          "returns error when --customer-id is missing",
			args:          []string{"time-entry", "list"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--customer-id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestTimeEntryCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantProfileID != "" && tc.svc.listProfileID != tc.wantProfileID {
				t.Errorf("ListByCustomerProfile() customerID = %q, want %q", tc.svc.listProfileID, tc.wantProfileID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- time-entry list-unbilled --

func TestTimeEntryListUnbilledCLI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		svc            *stubTimeEntryService
		wantErr        bool
		wantErrSubstr  string
		wantContain    string
		wantCustomerID string
	}{
		{
			name: "lists unbilled entries for customer profile",
			args: []string{"time-entry", "list-unbilled", "--customer-id=cus_1"},
			svc: &stubTimeEntryService{
				listUnbilledRes: []app.TimeEntryDTO{
					{ID: "te_ub1", Description: "Unbilled work", Hours: 6000},
				},
			},
			wantCustomerID: "cus_1",
			wantContain:    "te_ub1",
		},
		{
			name:          "returns error when --customer-id is missing",
			args:          []string{"time-entry", "list-unbilled"},
			svc:           &stubTimeEntryService{},
			wantErr:       true,
			wantErrSubstr: "--customer-id",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			cmd := newTestTimeEntryCommand(tc.svc)
			err := cmd.Run(context.Background(), tc.args, &out)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Run() error = nil, want error")
				}
				if tc.wantErrSubstr != "" && !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("Run() error = %q, want substring %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if tc.wantCustomerID != "" && tc.svc.listUnbilledID != tc.wantCustomerID {
				t.Errorf("ListUnbilled() customerID = %q, want %q", tc.svc.listUnbilledID, tc.wantCustomerID)
			}
			if tc.wantContain != "" && !strings.Contains(out.String(), tc.wantContain) {
				t.Errorf("Run() output = %q, want contains %q", out.String(), tc.wantContain)
			}
		})
	}
}

// -- unknown subcommand --

func TestTimeEntryUnknownSubcommand(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	cmd := newTestTimeEntryCommand(&stubTimeEntryService{})
	err := cmd.Run(context.Background(), []string{"time-entry", "bogus"}, &out)
	if err == nil {
		t.Fatal("Run() error = nil, want error for unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Errorf("Run() error = %q, want contains 'unknown'", err.Error())
	}
}
