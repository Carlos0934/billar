package app

import (
	"context"
	"errors"
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
