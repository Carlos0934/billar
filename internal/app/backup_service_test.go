package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type backupSnapshotterStub struct {
	record BackupRecord
	err    error
	dbPath string
	dir    string
}

func (s *backupSnapshotterStub) Create(ctx context.Context, dbPath, destDir string) (BackupRecord, error) {
	s.dbPath = dbPath
	s.dir = destDir
	return s.record, s.err
}

type backupListerStub struct {
	records []BackupRecord
	err     error
	dir     string
}

func (l *backupListerStub) List(ctx context.Context, dir string) ([]BackupRecord, error) {
	l.dir = dir
	return l.records, l.err
}

type backupPathsStub struct {
	paths RuntimePaths
	err   error
}

func (p backupPathsStub) RuntimePaths(context.Context) (RuntimePaths, error) { return p.paths, p.err }

func TestBackupServiceCreateMapsRecordAndWarnings(t *testing.T) {
	createdAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	snap := &backupSnapshotterStub{record: BackupRecord{
		ID:            "backup-1",
		File:          "/backups/backup-1.db",
		SidecarFile:   "/backups/backup-1.db.json",
		CreatedAt:     createdAt,
		SchemaVersion: 3,
		SizeBytes:     42,
		SHA256:        "abc123",
		SourceDBPath:  "/data/billar.db",
		Metadata:      true,
	}}
	svc := NewBackupService(snap, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{
		DBPath:    RuntimePath{Path: "/data/billar.db", Source: "configured"},
		BackupDir: RuntimePath{Path: "/backups", Source: "configured"},
	}})

	got, err := svc.Create(context.Background())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snap.dbPath != "/data/billar.db" || snap.dir != "/backups" {
		t.Fatalf("snapshot called with db=%q dir=%q", snap.dbPath, snap.dir)
	}
	if got.ID != "backup-1" || got.File != "/backups/backup-1.db" || got.SHA256 != "abc123" || got.SizeBytes != 42 || !got.Metadata {
		t.Fatalf("Create() = %+v, want mapped record", got)
	}
	if len(got.Warnings) != 1 || !strings.Contains(strings.ToLower(got.Warnings[0]), "sensitive") {
		t.Fatalf("Warnings = %v, want sensitive data warning", got.Warnings)
	}
}

func TestBackupServiceCreateWrapsOverwriteError(t *testing.T) {
	overwrite := errors.New("/backups/existing.db already exists")
	svc := NewBackupService(&backupSnapshotterStub{err: overwrite}, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{
		DBPath:    RuntimePath{Path: "/data/billar.db"},
		BackupDir: RuntimePath{Path: "/backups"},
	}})

	_, err := svc.Create(context.Background())
	if err == nil || !strings.Contains(err.Error(), "create backup") || !strings.Contains(err.Error(), overwrite.Error()) {
		t.Fatalf("Create() error = %v, want wrapped overwrite error", err)
	}
}

func TestBackupServiceListMapsRecordsAndPreservesOrdering(t *testing.T) {
	newer := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	older := newer.Add(-time.Hour)
	lister := &backupListerStub{records: []BackupRecord{
		{ID: "new", File: "/backups/new.db", CreatedAt: newer, Metadata: true, SHA256: "newsha"},
		{ID: "old", File: "/backups/old.db", CreatedAt: older, Metadata: false},
	}}
	svc := NewBackupService(&backupSnapshotterStub{}, lister, backupPathsStub{paths: RuntimePaths{BackupDir: RuntimePath{Path: "/backups"}}})

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if lister.dir != "/backups" || got.Dir != "/backups" {
		t.Fatalf("List dirs = lister %q dto %q, want /backups", lister.dir, got.Dir)
	}
	if len(got.Backups) != 2 || got.Backups[0].ID != "new" || got.Backups[1].ID != "old" || got.Backups[1].Metadata {
		t.Fatalf("List() = %+v, want ordering preserved and metadata mapped", got)
	}

	emptySvc := NewBackupService(&backupSnapshotterStub{}, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{BackupDir: RuntimePath{Path: "/empty"}}})
	empty, err := emptySvc.List(context.Background())
	if err != nil {
		t.Fatalf("empty List() error = %v", err)
	}
	if empty.Backups == nil || len(empty.Backups) != 0 {
		t.Fatalf("empty List().Backups = %#v, want empty non-nil slice", empty.Backups)
	}
}

func TestBackupServicePropagatesPathAndListErrors(t *testing.T) {
	pathErr := errors.New("paths unavailable")
	svc := NewBackupService(&backupSnapshotterStub{}, &backupListerStub{}, backupPathsStub{err: pathErr})
	if _, err := svc.Create(context.Background()); err == nil || !strings.Contains(err.Error(), pathErr.Error()) {
		t.Fatalf("Create() path error = %v, want propagated", err)
	}

	listErr := errors.New("read failed")
	svc = NewBackupService(&backupSnapshotterStub{}, &backupListerStub{err: listErr}, backupPathsStub{paths: RuntimePaths{BackupDir: RuntimePath{Path: "/backups"}}})
	if _, err := svc.List(context.Background()); err == nil || !strings.Contains(err.Error(), listErr.Error()) {
		t.Fatalf("List() error = %v, want propagated list error", err)
	}
}

func TestBackupRestoreDTOContracts(t *testing.T) {
	t.Parallel()

	request := BackupRestoreRequest{BackupID: "bk_1", DryRun: true, Force: false}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Marshal(BackupRestoreRequest) error = %v", err)
	}
	if got := string(encoded); !strings.Contains(got, `"backup_id":"bk_1"`) || !strings.Contains(got, `"dry_run":true`) || !strings.Contains(got, `"force":false`) {
		t.Fatalf("BackupRestoreRequest JSON = %s, want stable machine fields", got)
	}

	assertJSONAndTOONTag(t, reflect.TypeOf(BackupRestoreResultDTO{}), "TargetDBPath", "target_db_path")
	assertJSONAndTOONTag(t, reflect.TypeOf(BackupRestoreResultDTO{}), "SafetySnapshot", "safety_snapshot")
	assertJSONAndTOONTag(t, reflect.TypeOf(BackupValidation{}), "HashOK", "hash_ok")
	assertJSONAndTOONTag(t, reflect.TypeOf(BackupSafetySnapshot{}), "Skipped", "skipped")
}

func TestNewBackupServiceWithRestoreWiresSeams(t *testing.T) {
	t.Parallel()

	restorer := backupRestorerStub{}
	schema := backupSchemaProviderStub{version: 4}
	svc := NewBackupServiceWithRestore(&backupSnapshotterStub{}, &backupListerStub{}, backupPathsStub{}, restorer, schema)
	if svc.restore == nil {
		t.Fatal("restore seam = nil, want configured restorer")
	}
	if svc.schema == nil {
		t.Fatal("schema seam = nil, want configured schema provider")
	}
}

func TestBackupServiceRestoreOrchestratesPlanAndDestructivePaths(t *testing.T) {
	createdAt := time.Date(2026, 5, 1, 16, 0, 0, 0, time.UTC)
	record := BackupRecord{
		ID:            "bk_123",
		File:          "/backups/bk_123.db",
		SidecarFile:   "/backups/bk_123.db.json",
		CreatedAt:     createdAt,
		SchemaVersion: 7,
		SizeBytes:     4096,
		SHA256:        "abc123",
		SourceDBPath:  "/runtime/billar.db",
		Metadata:      true,
	}
	validation := BackupValidation{OK: true, SidecarOK: true, HashOK: true, SizeOK: true, IntegrityOK: true, TablesOK: true, SchemaOK: true, BackupSchema: 7, BinarySchema: 9}

	t.Run("dry run by id validates and skips snapshot and replace", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "billar.db")
		restorer := &configurableBackupRestorer{record: record, validation: validation}
		snapshotter := &backupSnapshotterStub{record: BackupRecord{ID: "safety"}}
		svc := NewBackupServiceWithRestore(snapshotter, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: target}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		got, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", DryRun: true})
		if err != nil {
			t.Fatalf("Restore() error = %v", err)
		}
		if !got.DryRun || got.Replaced || got.Backup.ID != "bk_123" || got.TargetDBPath != target || !got.Validation.OK {
			t.Fatalf("Restore() = %+v, want dry-run plan for selected backup", got)
		}
		if restorer.resolveReq.BackupID != "bk_123" || restorer.resolveDir != "/backups" || restorer.validateBinarySchema != 9 || restorer.validateTarget != target {
			t.Fatalf("restorer calls = req %+v dir %q target %q schema %d, want id lookup, target, binary schema", restorer.resolveReq, restorer.resolveDir, restorer.validateTarget, restorer.validateBinarySchema)
		}
		if snapshotter.dbPath != "" || restorer.replaceCalled {
			t.Fatalf("dry run snapshot db=%q replace=%t, want neither", snapshotter.dbPath, restorer.replaceCalled)
		}
	})

	t.Run("dry run by file accepts file basename as selector", func(t *testing.T) {
		fileRecord := record
		fileRecord.File = "/portable/bk_file.db"
		restorer := &configurableBackupRestorer{record: fileRecord, validation: validation}
		svc := NewBackupServiceWithRestore(&backupSnapshotterStub{}, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: "/runtime/billar.db"}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		got, err := svc.Restore(context.Background(), BackupRestoreRequest{File: "/portable/bk_file.db", DryRun: true})
		if err != nil {
			t.Fatalf("Restore(file dry-run) error = %v", err)
		}
		if got.Backup.File != "/portable/bk_file.db" || restorer.resolveReq.File != "/portable/bk_file.db" || got.Replaced {
			t.Fatalf("Restore(file dry-run) = %+v, resolve req %+v; want file-sourced plan", got, restorer.resolveReq)
		}
	})

	t.Run("destructive restore requires confirm unless forced", func(t *testing.T) {
		restorer := &configurableBackupRestorer{record: record, validation: validation}
		svc := NewBackupServiceWithRestore(&backupSnapshotterStub{}, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: "/runtime/billar.db"}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		_, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123"})
		if err == nil || !strings.Contains(err.Error(), "missing_confirmation") {
			t.Fatalf("Restore() error = %v, want missing confirmation", err)
		}
		if restorer.replaceCalled {
			t.Fatal("Replace called without confirmation")
		}

		if _, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Force: true}); err != nil {
			t.Fatalf("Restore(force) error = %v", err)
		}
		if !restorer.replaceCalled {
			t.Fatal("Restore(force) did not replace")
		}
	})

	t.Run("validation failure warns before snapshot on destructive attempt", func(t *testing.T) {
		validateErr := errors.New("hash_mismatch: computed hash differs")
		restorer := &configurableBackupRestorer{record: record, validateErr: validateErr}
		snapshotter := &backupSnapshotterStub{}
		svc := NewBackupServiceWithRestore(snapshotter, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: "/runtime/billar.db"}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		got, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Force: true})
		if err == nil || !strings.Contains(err.Error(), "validate backup") || !strings.Contains(err.Error(), "hash_mismatch") {
			t.Fatalf("Restore() error = %v, want wrapped validation error", err)
		}
		if !containsSubstring(got.Warnings, "concurrent_processes") {
			t.Fatalf("Restore() warnings = %v, want concurrent_processes on destructive validation failure", got.Warnings)
		}
		if snapshotter.dbPath != "" || restorer.replaceCalled {
			t.Fatalf("validation failure snapshot db=%q replace=%t, want neither", snapshotter.dbPath, restorer.replaceCalled)
		}
	})

	t.Run("snapshot failure stops before replace", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "billar.db")
		if err := os.WriteFile(target, []byte("live"), 0o600); err != nil {
			t.Fatalf("seed target: %v", err)
		}
		snapshotErr := errors.New("snapshot disk full")
		restorer := &configurableBackupRestorer{record: withSourceDBPath(record, target), validation: validation}
		snapshotter := &backupSnapshotterStub{err: snapshotErr}
		svc := NewBackupServiceWithRestore(snapshotter, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: target}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		_, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Confirm: "bk_123"})
		if err == nil || !strings.Contains(err.Error(), "create safety snapshot") || !strings.Contains(err.Error(), snapshotErr.Error()) {
			t.Fatalf("Restore() error = %v, want snapshot failure", err)
		}
		if snapshotter.dbPath != target || snapshotter.dir != "/backups" || restorer.replaceCalled {
			t.Fatalf("snapshot db=%q dir=%q replace=%t, want snapshot before no replace", snapshotter.dbPath, snapshotter.dir, restorer.replaceCalled)
		}
	})

	t.Run("replace failure reports preserved safety snapshot and rollback warning", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "billar.db")
		if err := os.WriteFile(target, []byte("live"), 0o600); err != nil {
			t.Fatalf("seed target: %v", err)
		}
		safety := BackupRecord{ID: "safety", File: "/backups/safety.db", Metadata: true}
		restorer := &configurableBackupRestorer{record: withSourceDBPath(record, target), validation: validation, replaceErr: errors.New("rename failed")}
		svc := NewBackupServiceWithRestore(&backupSnapshotterStub{record: safety}, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: target}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		got, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Confirm: "bk_123"})
		if err == nil || !strings.Contains(err.Error(), "replace target database") {
			t.Fatalf("Restore() error = %v, want replace failure", err)
		}
		if got.SafetySnapshot == nil || got.SafetySnapshot.Record == nil || got.SafetySnapshot.Record.ID != "safety" {
			t.Fatalf("Restore() safety snapshot = %+v, want preserved safety snapshot on failure", got.SafetySnapshot)
		}
		if !containsSubstring(got.Warnings, "rollback") || !containsSubstring(got.Warnings, "concurrent_processes") {
			t.Fatalf("Restore() warnings = %v, want rollback and concurrent-process warnings", got.Warnings)
		}
	})

	t.Run("success with live DB creates snapshot and replaces", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "billar.db")
		if err := os.WriteFile(target, []byte("live"), 0o600); err != nil {
			t.Fatalf("seed target: %v", err)
		}
		safety := BackupRecord{ID: "safety", File: "/backups/safety.db", Metadata: true}
		restorer := &configurableBackupRestorer{record: withSourceDBPath(record, target), validation: validation}
		snapshotter := &backupSnapshotterStub{record: safety}
		svc := NewBackupServiceWithRestore(snapshotter, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: target}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		got, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Confirm: "bk_123"})
		if err != nil {
			t.Fatalf("Restore() error = %v", err)
		}
		if !got.Replaced || got.SafetySnapshot == nil || got.SafetySnapshot.Record == nil || got.SafetySnapshot.Record.ID != "safety" || !containsSubstring(got.Warnings, "concurrent_processes") {
			t.Fatalf("Restore() = %+v, want replaced with safety snapshot and warning", got)
		}
		if snapshotter.dbPath != target || restorer.replaceTarget != target {
			t.Fatalf("snapshot target=%q replace target=%q, want %q", snapshotter.dbPath, restorer.replaceTarget, target)
		}
	})

	t.Run("success without live DB warns and emits null safety snapshot", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "missing.db")
		restorer := &configurableBackupRestorer{record: withSourceDBPath(record, target), validation: validation}
		snapshotter := &backupSnapshotterStub{record: BackupRecord{ID: "unexpected"}}
		svc := NewBackupServiceWithRestore(snapshotter, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: target}, BackupDir: RuntimePath{Path: "/backups"}}}, restorer, backupSchemaProviderStub{version: 9})

		got, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Confirm: "bk_123"})
		if err != nil {
			t.Fatalf("Restore() error = %v", err)
		}
		if !got.Replaced || got.SafetySnapshot != nil || !containsSubstring(got.Warnings, "no_live_db") {
			t.Fatalf("Restore() = %+v, want nil safety snapshot and no-live-db warning", got)
		}
		encoded, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("Marshal(Restore result) error = %v", err)
		}
		if !strings.Contains(string(encoded), `"safety_snapshot":null`) {
			t.Fatalf("Restore() JSON = %s, want safety_snapshot:null", encoded)
		}
		if snapshotter.dbPath != "" || restorer.replaceTarget != target {
			t.Fatalf("snapshot db=%q replace target=%q, want no snapshot and replace target", snapshotter.dbPath, restorer.replaceTarget)
		}
	})

	t.Run("selector and source target mismatch gates", func(t *testing.T) {
		mismatchedRecord := record
		mismatchedRecord.SourceDBPath = "/other-machine/billar.db"
		svc := NewBackupServiceWithRestore(&backupSnapshotterStub{}, &backupListerStub{}, backupPathsStub{paths: RuntimePaths{DBPath: RuntimePath{Path: "/runtime/billar.db"}, BackupDir: RuntimePath{Path: "/backups"}}}, &configurableBackupRestorer{record: mismatchedRecord, validation: validation}, backupSchemaProviderStub{version: 9})

		_, err := svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", File: "/backups/bk_123.db", DryRun: true})
		if err == nil || !strings.Contains(err.Error(), "exactly_one_selector") {
			t.Fatalf("Restore(both selectors) error = %v, want exactly_one_selector", err)
		}

		_, err = svc.Restore(context.Background(), BackupRestoreRequest{BackupID: "bk_123", Confirm: "bk_123"})
		if err == nil || !strings.Contains(err.Error(), "source_target_mismatch") {
			t.Fatalf("Restore(source mismatch) error = %v, want force gate", err)
		}
	})
}

type backupRestorerStub struct{}

func (backupRestorerStub) Resolve(context.Context, BackupRestoreRequest, string) (BackupRecord, error) {
	return BackupRecord{}, nil
}
func (backupRestorerStub) Validate(context.Context, BackupRecord, string, int) (BackupValidation, error) {
	return BackupValidation{}, nil
}
func (backupRestorerStub) Replace(context.Context, BackupRecord, string) error { return nil }

type backupSchemaProviderStub struct{ version int }

func (s backupSchemaProviderStub) LatestSchemaVersion() int { return s.version }

type configurableBackupRestorer struct {
	record               BackupRecord
	validation           BackupValidation
	resolveErr           error
	validateErr          error
	replaceErr           error
	resolveReq           BackupRestoreRequest
	resolveDir           string
	validateTarget       string
	validateBinarySchema int
	replaceCalled        bool
	replaceTarget        string
}

func (r *configurableBackupRestorer) Resolve(ctx context.Context, req BackupRestoreRequest, backupDir string) (BackupRecord, error) {
	_ = ctx
	r.resolveReq = req
	r.resolveDir = backupDir
	return r.record, r.resolveErr
}

func (r *configurableBackupRestorer) Validate(ctx context.Context, record BackupRecord, targetDBPath string, binarySchema int) (BackupValidation, error) {
	_ = ctx
	_ = record
	r.validateTarget = targetDBPath
	r.validateBinarySchema = binarySchema
	return r.validation, r.validateErr
}

func (r *configurableBackupRestorer) Replace(ctx context.Context, record BackupRecord, targetDBPath string) error {
	_ = ctx
	_ = record
	r.replaceCalled = true
	r.replaceTarget = targetDBPath
	return r.replaceErr
}

func withSourceDBPath(record BackupRecord, sourceDBPath string) BackupRecord {
	record.SourceDBPath = sourceDBPath
	return record
}

func containsSubstring(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func assertJSONAndTOONTag(t *testing.T, typ reflect.Type, fieldName, want string) {
	t.Helper()
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		t.Fatalf("%s.%s field missing", typ.Name(), fieldName)
	}
	if got := field.Tag.Get("json"); got != want {
		t.Fatalf("%s.%s json tag = %q, want %q", typ.Name(), fieldName, got, want)
	}
	if got := field.Tag.Get("toon"); got != want {
		t.Fatalf("%s.%s toon tag = %q, want %q", typ.Name(), fieldName, got, want)
	}
}
