package main

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Carlos0934/billar/internal/infra/config"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func TestNewCommandWiresTimeEntryService(t *testing.T) {
	t.Parallel()

	store := mustOpenCLIStore(t)
	seedCLIWiringFixture(t, store.DB())

	cmd := newCommand(config.Config{AppName: "billar", ColorEnabled: false}, store)

	var out bytes.Buffer
	if err := cmd.Run(context.Background(), []string{"time-entry", "record", `--json={"customer_profile_id":"cus_cli_wiring","service_agreement_id":"sa_cli_wiring","description":"wiring check","hours":60,"billable":true,"date":"2026-04-10T00:00:00Z"}`, "--format", "json"}, &out); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "wiring check") {
		t.Fatalf("Run() output = %q, want wiring check payload", out.String())
	}

	timeEntryStore := infrasqlite.NewTimeEntryStore(store)
	entries, err := timeEntryStore.ListByCustomerProfile(context.Background(), "cus_cli_wiring")
	if err != nil {
		t.Fatalf("ListByCustomerProfile() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ListByCustomerProfile() len = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.CustomerProfileID != "cus_cli_wiring" || got.ServiceAgreementID != "sa_cli_wiring" {
		t.Fatalf("ListByCustomerProfile() = %+v, want seeded wiring entry", got)
	}
	if got.Description != "wiring check" {
		t.Fatalf("ListByCustomerProfile() description = %q, want %q", got.Description, "wiring check")
	}
}

func TestMainRunsHealthCommand(t *testing.T) {
	storePath := t.TempDir() + "/cli-main.db"
	t.Setenv("BILLAR_DB_PATH", storePath)

	oldArgs := os.Args
	os.Args = []string{"billar", "health"}
	t.Cleanup(func() { os.Args = oldArgs })

	main()
}

func mustOpenCLIStore(t *testing.T) *infrasqlite.Store {
	t.Helper()

	store, err := infrasqlite.Open(t.TempDir() + "/cli-entrypoint.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}

func seedCLIWiringFixture(t *testing.T, db *sql.DB) {
	t.Helper()

	now := time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC).UnixNano()
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO legal_entities (id, type, legal_name, trade_name, tax_id, email, phone, website, billing_address, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"le_cli_wiring", "company", "CLI Wiring Co", "", "", "", "", "", "{}", now, now); err != nil {
		t.Fatalf("insert legal entity: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, notes, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"cus_cli_wiring", "le_cli_wiring", "active", "USD", "", now, now); err != nil {
		t.Fatalf("insert customer profile: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `
INSERT INTO service_agreements (id, customer_profile_id, name, description, billing_mode, hourly_rate, currency, active, valid_from, valid_until, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"sa_cli_wiring", "cus_cli_wiring", "CLI Wiring Agreement", "", "hourly", 5000, "USD", 1, nil, nil, now, now); err != nil {
		t.Fatalf("insert service agreement: %v", err)
	}
}
