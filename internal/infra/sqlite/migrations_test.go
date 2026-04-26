package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

func newTempDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "migrations.db"))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	cleanup := func() {
		if err := db.Close(); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	}
	return db, cleanup
}

func TestLoadMigrations_OrdersByVersion(t *testing.T) {
	t.Parallel()

	migrations, err := loadMigrations(fstest.MapFS{
		"0002_b.sql": {Data: []byte("CREATE TABLE b (id TEXT);")},
		"0001_a.sql": {Data: []byte("CREATE TABLE a (id TEXT);")},
	})
	if err != nil {
		t.Fatalf("loadMigrations() error = %v", err)
	}

	if got, want := len(migrations), 2; got != want {
		t.Fatalf("len(migrations) = %d, want %d", got, want)
	}
	if got, want := migrations[0].version, 1; got != want {
		t.Fatalf("first migration version = %d, want %d", got, want)
	}
	if got, want := migrations[0].name, "0001_a"; got != want {
		t.Fatalf("first migration name = %q, want %q", got, want)
	}
	if got, want := migrations[1].version, 2; got != want {
		t.Fatalf("second migration version = %d, want %d", got, want)
	}
}

func TestLoadMigrations_RejectsDuplicateAndGap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		source  fs.FS
		wantErr string
	}{
		{
			name: "duplicate version",
			source: fstest.MapFS{
				"0001_a.sql": {Data: []byte("CREATE TABLE a (id TEXT);")},
				"0001_b.sql": {Data: []byte("CREATE TABLE b (id TEXT);")},
			},
			wantErr: "duplicate migration version 1",
		},
		{
			name: "gap",
			source: fstest.MapFS{
				"0001_a.sql": {Data: []byte("CREATE TABLE a (id TEXT);")},
				"0003_c.sql": {Data: []byte("CREATE TABLE c (id TEXT);")},
			},
			wantErr: "migration version gap: got 3 after 1",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := loadMigrations(tt.source)
			if err == nil {
				t.Fatal("loadMigrations() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("loadMigrations() error = %q, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestApplyMigrations_FreshDBBaseline(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)

	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	version, name, appliedAt := migrationRow(t, db, 1)
	if version != 1 || name != "0001_baseline" || appliedAt <= 0 {
		t.Fatalf("schema_migrations row = (%d, %q, %d), want (1, 0001_baseline, >0)", version, name, appliedAt)
	}
	assertTableExists(t, db, "legal_entities")
	assertTableExists(t, db, "customer_profiles")
}

func TestApplyMigrations_AddsInvoiceMetadataColumns(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)

	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	columns := invoiceColumnInfo(t, db)
	for _, name := range []string{"period_start", "period_end", "due_date"} {
		col, ok := columns[name]
		if !ok {
			t.Fatalf("column %q missing from invoices", name)
		}
		if col.notNull {
			t.Fatalf("column %q notnull = true, want nullable", name)
		}
	}
	notes, ok := columns["notes"]
	if !ok {
		t.Fatal("column notes missing from invoices")
	}
	if !notes.notNull || notes.defaultValue != "''" {
		t.Fatalf("notes column = %+v, want NOT NULL DEFAULT ''", notes)
	}
}

func TestApplyMigrations_AddsManualInvoiceLineSnapshots(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)

	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	columns := invoiceLineColumnInfo(t, db)
	timeEntryID, ok := columns["time_entry_id"]
	if !ok {
		t.Fatal("column time_entry_id missing from invoice_lines")
	}
	if timeEntryID.notNull {
		t.Fatalf("time_entry_id notnull = true, want nullable for manual lines")
	}
	description, ok := columns["description"]
	if !ok {
		t.Fatal("column description missing from invoice_lines")
	}
	if description.typ != "TEXT" || !description.notNull || description.defaultValue != "''" {
		t.Fatalf("description column = %+v, want TEXT NOT NULL DEFAULT ''", description)
	}
	quantityMin, ok := columns["quantity_min"]
	if !ok {
		t.Fatal("column quantity_min missing from invoice_lines")
	}
	if quantityMin.typ != "INTEGER" || !quantityMin.notNull || quantityMin.defaultValue != "0" {
		t.Fatalf("quantity_min column = %+v, want INTEGER NOT NULL DEFAULT 0", quantityMin)
	}
}

func TestApplyMigrations_BackfillsInvoiceLineSnapshots(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)
	if _, err := db.Exec(legacySchemaSQLForMigrationTests); err != nil {
		t.Fatalf("exec legacy schema: %v", err)
	}
	seedLegacyInvoiceLineForSnapshotMigration(t, db)

	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	var description string
	var quantityMin int64
	if err := db.QueryRow(`SELECT description, quantity_min FROM invoice_lines WHERE id = 'inl_existing'`).Scan(&description, &quantityMin); err != nil {
		t.Fatalf("query backfilled invoice line: %v", err)
	}
	if description != "Legacy consulting" {
		t.Fatalf("description = %q, want Legacy consulting", description)
	}
	if quantityMin != 90 {
		t.Fatalf("quantity_min = %d, want 90", quantityMin)
	}
}

func TestApplyMigrations_BackStampsExistingDB(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)
	if _, err := db.Exec(legacySchemaSQLForMigrationTests); err != nil {
		t.Fatalf("exec legacy schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO legal_entities (id, type, legal_name, created_at, updated_at) VALUES ('le_existing', 'company', 'Existing Co', 11, 22)`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("applyMigrations() error = %v", err)
	}

	_, name, _ := migrationRow(t, db, 1)
	if name != "0001_baseline" {
		t.Fatalf("baseline migration name = %q, want 0001_baseline", name)
	}
	var legalName string
	if err := db.QueryRow(`SELECT legal_name FROM legal_entities WHERE id = 'le_existing'`).Scan(&legalName); err != nil {
		t.Fatalf("query preserved legal entity: %v", err)
	}
	if legalName != "Existing Co" {
		t.Fatalf("preserved legal entity legal_name = %q, want Existing Co", legalName)
	}
}

func TestApplyMigrations_RejectsUnknownLegacy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup string
	}{
		{
			name:  "unrelated table only",
			setup: `CREATE TABLE unrelated (id TEXT PRIMARY KEY);`,
		},
		{
			name: "missing baseline tables",
			setup: `
CREATE TABLE legal_entities (id TEXT PRIMARY KEY);
CREATE TABLE customer_profiles (id TEXT PRIMARY KEY);`,
		},
		{
			name:  "current baseline plus unrelated table",
			setup: legacySchemaSQLForMigrationTests + `CREATE TABLE unrelated (id TEXT PRIMARY KEY);`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, cleanup := newTempDB(t)
			t.Cleanup(cleanup)
			if _, err := db.Exec(tt.setup); err != nil {
				t.Fatalf("setup legacy schema: %v", err)
			}

			err := applyMigrations(db, migrationsFS)
			if err == nil {
				t.Fatal("applyMigrations() error = nil, want baseline mismatch")
			}
			if !strings.Contains(err.Error(), "baseline detection") || !strings.Contains(err.Error(), "baseline mismatch") {
				t.Fatalf("applyMigrations() error = %q, want baseline detection mismatch context", err)
			}
			assertNoMigrationRows(t, db)
		})
	}
}

func TestApplyMigrations_AppliesMultiplePendingMigrationsInOrder(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)

	baselineOnly := fstest.MapFS{
		"0001_baseline.sql": {Data: []byte(`CREATE TABLE ordered_steps (step INTEGER PRIMARY KEY, label TEXT NOT NULL);`)},
	}
	if err := applyMigrations(db, baselineOnly); err != nil {
		t.Fatalf("baseline applyMigrations() error = %v", err)
	}

	withPending := fstest.MapFS{
		"0001_baseline.sql": {Data: []byte(`CREATE TABLE ordered_steps (step INTEGER PRIMARY KEY, label TEXT NOT NULL);`)},
		"0002_second.sql":   {Data: []byte(`INSERT INTO ordered_steps (step, label) VALUES (2, 'second');`)},
		"0003_third.sql":    {Data: []byte(`INSERT INTO ordered_steps (step, label) SELECT 3, label || ' then third' FROM ordered_steps WHERE step = 2;`)},
	}
	if err := applyMigrations(db, withPending); err != nil {
		t.Fatalf("pending applyMigrations() error = %v", err)
	}

	assertMigrationVersions(t, db, []int{1, 2, 3})
	assertMigrationNames(t, db, []string{"0001_baseline", "0002_second", "0003_third"})
	var label string
	if err := db.QueryRow(`SELECT label FROM ordered_steps WHERE step = 3`).Scan(&label); err != nil {
		t.Fatalf("query ordered step 3: %v", err)
	}
	if label != "second then third" {
		t.Fatalf("ordered step 3 label = %q, want %q", label, "second then third")
	}
}

func TestApplyMigrations_IdempotentReopen(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)

	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("first applyMigrations() error = %v", err)
	}
	_, _, firstAppliedAt := migrationRow(t, db, 1)
	if err := applyMigrations(db, migrationsFS); err != nil {
		t.Fatalf("second applyMigrations() error = %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != 3 {
		t.Fatalf("schema_migrations row count = %d, want 3", count)
	}
	_, _, secondAppliedAt := migrationRow(t, db, 1)
	if secondAppliedAt != firstAppliedAt {
		t.Fatalf("baseline applied_at changed on reopen: got %d, want %d", secondAppliedAt, firstAppliedAt)
	}
}

func TestApplyMigrations_FailureRollback(t *testing.T) {
	t.Parallel()

	db, cleanup := newTempDB(t)
	t.Cleanup(cleanup)

	err := applyMigrations(db, fstest.MapFS{
		"0001_baseline.sql": {Data: []byte(legacySchemaSQLForMigrationTests)},
		"0002_bad.sql":      {Data: []byte("CREATE TABLE migration_two_partial (id TEXT); INSERT INTO missing_table VALUES (1);")},
		"0003_later.sql":    {Data: []byte("CREATE TABLE should_not_exist (id TEXT);")},
	})
	if err == nil {
		t.Fatal("applyMigrations() error = nil, want version 2 failure")
	}
	if !strings.Contains(err.Error(), "apply migration 2") {
		t.Fatalf("applyMigrations() error = %q, want version 2 context", err)
	}
	if errors.Unwrap(err) == nil {
		t.Fatalf("applyMigrations() error = %q, want wrapped driver error", err)
	}
	assertMigrationVersions(t, db, []int{1})
	assertTableMissing(t, db, "migration_two_partial")
	assertTableMissing(t, db, "should_not_exist")
}

func TestApplyMigrations_ErrorPhaseContext(t *testing.T) {
	t.Parallel()

	t.Run("load phase", func(t *testing.T) {
		t.Parallel()
		db, cleanup := newTempDB(t)
		t.Cleanup(cleanup)

		err := applyMigrations(db, fstest.MapFS{"0002_gap.sql": {Data: []byte("CREATE TABLE gap (id TEXT);")}})
		if err == nil {
			t.Fatal("applyMigrations() error = nil, want load failure")
		}
		if !strings.Contains(err.Error(), "load migrations") {
			t.Fatalf("applyMigrations() error = %q, want load migrations context", err)
		}
	})

	t.Run("apply phase wraps driver error", func(t *testing.T) {
		t.Parallel()
		db, cleanup := newTempDB(t)
		t.Cleanup(cleanup)

		err := applyMigrations(db, fstest.MapFS{"0001_baseline.sql": {Data: []byte("INSERT INTO missing_table VALUES (1);")}})
		if err == nil {
			t.Fatal("applyMigrations() error = nil, want SQL failure")
		}
		if !strings.Contains(err.Error(), "apply migration 1") {
			t.Fatalf("applyMigrations() error = %q, want apply migration 1 context", err)
		}
		if errors.Unwrap(err) == nil {
			t.Fatalf("applyMigrations() error = %q, want wrapped driver error", err)
		}
	})
}

func migrationRow(t *testing.T, db *sql.DB, version int) (int, string, int64) {
	t.Helper()

	var gotVersion int
	var name string
	var appliedAt int64
	err := db.QueryRowContext(context.Background(), `SELECT version, name, applied_at FROM schema_migrations WHERE version = ?`, version).Scan(&gotVersion, &name, &appliedAt)
	if err != nil {
		t.Fatalf("query schema_migrations version %d: %v", version, err)
	}
	return gotVersion, name, appliedAt
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	err := db.QueryRowContext(context.Background(), `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if err != nil {
		t.Fatalf("table %q missing: %v", table, err)
	}
	if name != table {
		t.Fatalf("table name = %q, want %q", name, table)
	}
}

func assertTableMissing(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	err := db.QueryRowContext(context.Background(), `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("table %q query error = %v, want sql.ErrNoRows", table, err)
	}
}

type sqliteColumnInfo struct {
	typ          string
	notNull      bool
	defaultValue string
}

func invoiceColumnInfo(t *testing.T, db *sql.DB) map[string]sqliteColumnInfo {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(invoices)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(invoices): %v", err)
	}
	defer rows.Close()
	columns := map[string]sqliteColumnInfo{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan column info: %v", err)
		}
		columns[name] = sqliteColumnInfo{typ: typ, notNull: notnull == 1, defaultValue: dflt.String}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate column info: %v", err)
	}
	return columns
}

func invoiceLineColumnInfo(t *testing.T, db *sql.DB) map[string]sqliteColumnInfo {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(invoice_lines)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(invoice_lines): %v", err)
	}
	defer rows.Close()
	columns := map[string]sqliteColumnInfo{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan invoice line column info: %v", err)
		}
		columns[name] = sqliteColumnInfo{typ: typ, notNull: notnull == 1, defaultValue: dflt.String}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate invoice line column info: %v", err)
	}
	return columns
}

func seedLegacyInvoiceLineForSnapshotMigration(t *testing.T, db *sql.DB) {
	t.Helper()
	now := int64(1712534400000000000)
	statements := []string{
		`INSERT INTO legal_entities (id, type, legal_name, created_at, updated_at) VALUES ('le_existing', 'company', 'Existing Co', 11, 22)`,
		`INSERT INTO customer_profiles (id, legal_entity_id, status, default_currency, created_at, updated_at) VALUES ('cus_existing', 'le_existing', 'active', 'USD', 11, 22)`,
		`INSERT INTO service_agreements (id, customer_profile_id, name, billing_mode, hourly_rate, currency, active, created_at, updated_at) VALUES ('sa_existing', 'cus_existing', 'Support', 'hourly', 10000, 'USD', 1, 11, 22)`,
		`INSERT INTO time_entries (id, service_agreement_id, description, hours, billable, invoice_id, date, created_at, updated_at) VALUES ('te_existing', 'sa_existing', 'Legacy consulting', 15000, 1, 'inv_existing', 1712534400000000000, 11, 22)`,
		`INSERT INTO invoices (id, invoice_number, customer_id, status, currency, created_at, updated_at) VALUES ('inv_existing', '', 'cus_existing', 'draft', 'USD', 11, 22)`,
		`INSERT INTO invoice_lines (id, invoice_id, service_agreement_id, time_entry_id, unit_rate_amount, unit_rate_currency) VALUES ('inl_existing', 'inv_existing', 'sa_existing', 'te_existing', 10000, 'USD')`,
	}
	_ = now
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("seed legacy invoice line: %v", err)
		}
	}
}

func assertNoMigrationRows(t *testing.T, db *sql.DB) {
	t.Helper()

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return
	}
	if err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != 0 {
		t.Fatalf("schema_migrations row count = %d, want 0", count)
	}
}

func assertMigrationVersions(t *testing.T, db *sql.DB, want []int) {
	t.Helper()

	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query migration versions: %v", err)
	}
	defer rows.Close()

	var got []int
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			t.Fatalf("scan migration version: %v", err)
		}
		got = append(got, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migration versions: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("migration versions = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("migration versions = %v, want %v", got, want)
		}
	}
}

func assertMigrationNames(t *testing.T, db *sql.DB, want []string) {
	t.Helper()

	rows, err := db.Query(`SELECT name FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query migration names: %v", err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan migration name: %v", err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate migration names: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("migration names = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("migration names = %v, want %v", got, want)
		}
	}
}

const legacySchemaSQLForMigrationTests = `-- Legal entities hold shared identity and contact information.
-- Both issuers (billing operators) and customers reference a legal entity.
CREATE TABLE IF NOT EXISTS legal_entities (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    legal_name TEXT NOT NULL,
    trade_name TEXT NOT NULL DEFAULT '',
    tax_id TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    website TEXT NOT NULL DEFAULT '',
    billing_address TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_legal_entities_legal_name ON legal_entities(legal_name);
CREATE INDEX IF NOT EXISTS idx_legal_entities_created_at ON legal_entities(created_at);

-- Issuer profiles represent the billing operator (the user's own company).
-- There is typically one issuer profile per installation.
-- The UNIQUE constraint on legal_entity_id enforces 1:1 relationship.
CREATE TABLE IF NOT EXISTS issuer_profiles (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL UNIQUE,
    default_currency TEXT NOT NULL DEFAULT 'USD',
    default_notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (legal_entity_id) REFERENCES legal_entities(id) ON DELETE CASCADE
);

-- Customer profiles represent clients to be billed.
-- References a legal entity for identity/contact data.
-- The UNIQUE constraint on legal_entity_id enforces 1:1 relationship.
CREATE TABLE IF NOT EXISTS customer_profiles (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'USD',
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (legal_entity_id) REFERENCES legal_entities(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_customer_profiles_status ON customer_profiles(status);
CREATE INDEX IF NOT EXISTS idx_customer_profiles_created_at ON customer_profiles(created_at);

-- Service agreements define billing terms for a customer profile.
-- Multiple agreements can exist per customer profile (e.g. different projects or rates over time).
CREATE TABLE IF NOT EXISTS service_agreements (
    id TEXT PRIMARY KEY,
    customer_profile_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    billing_mode TEXT NOT NULL,
    hourly_rate INTEGER NOT NULL,
    currency TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    valid_from INTEGER,
    valid_until INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (customer_profile_id) REFERENCES customer_profiles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_service_agreements_customer_profile_id ON service_agreements(customer_profile_id);
CREATE INDEX IF NOT EXISTS idx_service_agreements_created_at ON service_agreements(created_at);

-- Time entries record units of work performed for a customer under a service agreement.
-- customer_profile_id is NOT stored here; it is always derived via JOIN on service_agreements.
CREATE TABLE IF NOT EXISTS time_entries (
    id TEXT PRIMARY KEY,
    service_agreement_id TEXT NOT NULL,
    description TEXT NOT NULL,
    hours INTEGER NOT NULL,
    billable INTEGER NOT NULL DEFAULT 1,
    invoice_id TEXT,
    date INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (service_agreement_id) REFERENCES service_agreements(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_time_entries_service_agreement_id ON time_entries(service_agreement_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_date ON time_entries(date);

-- Invoice sequences track the next available number for each calendar year.
-- The sequence is global (not per-customer) and resets automatically at year rollover.
CREATE TABLE IF NOT EXISTS invoice_sequences (
    year INTEGER PRIMARY KEY,
    next_seq INTEGER NOT NULL DEFAULT 1
);

-- Invoices represent billing documents issued to customers.
-- Status lifecycle: draft → issued → discarded (soft-delete for issued).
-- Draft invoices are hard-deleted on discard; issued invoices are soft-deleted (status=discarded).
CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    invoice_number TEXT NOT NULL DEFAULT '',
    customer_id TEXT NOT NULL,
    status TEXT NOT NULL,
    currency TEXT NOT NULL,
    issued_at INTEGER,
    discarded_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (customer_id) REFERENCES customer_profiles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_invoices_customer_id ON invoices(customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);

-- Invoice lines are the individual line items on an invoice.
-- Each line corresponds to a single time entry being billed.
CREATE TABLE IF NOT EXISTS invoice_lines (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    service_agreement_id TEXT NOT NULL,
    time_entry_id TEXT NOT NULL,
    unit_rate_amount INTEGER NOT NULL,
    unit_rate_currency TEXT NOT NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE,
    FOREIGN KEY (service_agreement_id) REFERENCES service_agreements(id) ON DELETE CASCADE,
    FOREIGN KEY (time_entry_id) REFERENCES time_entries(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_invoice_lines_invoice_id ON invoice_lines(invoice_id);
`
