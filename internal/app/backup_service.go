package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BackupSnapshotter interface {
	Create(ctx context.Context, dbPath, destDir string) (BackupRecord, error)
}

type BackupLister interface {
	List(ctx context.Context, dir string) ([]BackupRecord, error)
}

type BackupPathsProvider interface {
	RuntimePaths(ctx context.Context) (RuntimePaths, error)
}

type BackupRestorer interface {
	Resolve(ctx context.Context, req BackupRestoreRequest, backupDir string) (BackupRecord, error)
	Validate(ctx context.Context, record BackupRecord, targetDBPath string, binarySchema int) (BackupValidation, error)
	Replace(ctx context.Context, record BackupRecord, targetDBPath string) error
}

type BackupSchemaProvider interface {
	LatestSchemaVersion() int
}

type BackupService struct {
	snap    BackupSnapshotter
	list    BackupLister
	paths   BackupPathsProvider
	restore BackupRestorer
	schema  BackupSchemaProvider
}

func NewBackupService(snap BackupSnapshotter, list BackupLister, paths BackupPathsProvider) BackupService {
	return BackupService{snap: snap, list: list, paths: paths}
}

func NewBackupServiceWithRestore(snap BackupSnapshotter, list BackupLister, paths BackupPathsProvider, restore BackupRestorer, schema BackupSchemaProvider) BackupService {
	return BackupService{snap: snap, list: list, paths: paths, restore: restore, schema: schema}
}

func (s BackupService) Create(ctx context.Context) (BackupRecordDTO, error) {
	paths, err := s.paths.RuntimePaths(ctx)
	if err != nil {
		return BackupRecordDTO{}, fmt.Errorf("resolve backup paths: %w", err)
	}
	record, err := s.snap.Create(ctx, strings.TrimSpace(paths.DBPath.Path), strings.TrimSpace(paths.BackupDir.Path))
	if err != nil {
		return BackupRecordDTO{}, fmt.Errorf("create backup: %w", err)
	}
	dto := backupRecordDTO(record)
	dto.Warnings = sensitiveBackupWarnings()
	return dto, nil
}

func (s BackupService) List(ctx context.Context) (BackupListDTO, error) {
	paths, err := s.paths.RuntimePaths(ctx)
	if err != nil {
		return BackupListDTO{}, fmt.Errorf("resolve backup paths: %w", err)
	}
	dir := strings.TrimSpace(paths.BackupDir.Path)
	records, err := s.list.List(ctx, dir)
	if err != nil {
		return BackupListDTO{}, fmt.Errorf("list backups: %w", err)
	}
	dtos := make([]BackupRecordDTO, 0, len(records))
	for _, record := range records {
		dtos = append(dtos, backupRecordDTO(record))
	}
	return BackupListDTO{Dir: dir, Backups: dtos}, nil
}

func (s BackupService) Restore(ctx context.Context, req BackupRestoreRequest) (BackupRestoreResultDTO, error) {
	if err := validateRestoreSelector(req); err != nil {
		return BackupRestoreResultDTO{}, err
	}
	if s.restore == nil {
		return BackupRestoreResultDTO{}, fmt.Errorf("backup restore service is required")
	}
	if s.schema == nil {
		return BackupRestoreResultDTO{}, fmt.Errorf("backup schema provider is required")
	}

	paths, err := s.paths.RuntimePaths(ctx)
	if err != nil {
		return BackupRestoreResultDTO{}, fmt.Errorf("resolve backup paths: %w", err)
	}
	targetDBPath := strings.TrimSpace(paths.DBPath.Path)
	backupDir := strings.TrimSpace(paths.BackupDir.Path)

	record, err := s.restore.Resolve(ctx, req, backupDir)
	if err != nil {
		return BackupRestoreResultDTO{}, fmt.Errorf("resolve backup source: %w", err)
	}
	result := BackupRestoreResultDTO{
		Backup:       backupRecordDTO(record),
		TargetDBPath: targetDBPath,
		DryRun:       req.DryRun,
	}
	if !req.DryRun {
		result.Warnings = append(result.Warnings, concurrentRestoreWarning())
	}
	validation, err := s.restore.Validate(ctx, record, targetDBPath, s.schema.LatestSchemaVersion())
	result.Validation = validation
	if err != nil {
		return result, fmt.Errorf("validate backup: %w", err)
	}
	if req.DryRun {
		return result, nil
	}
	if err := requireRestoreConfirmation(req, record); err != nil {
		return result, err
	}
	if err := requireSourceTargetMatchUnlessForced(req, record, targetDBPath); err != nil {
		return result, err
	}

	targetExists, err := restorePathExists(targetDBPath)
	if err != nil {
		return result, fmt.Errorf("inspect target database: %w", err)
	}
	if targetExists {
		if s.snap == nil {
			return result, fmt.Errorf("backup snapshotter is required")
		}
		snapshot, err := s.snap.Create(ctx, targetDBPath, backupDir)
		if err != nil {
			return result, fmt.Errorf("create safety snapshot: %w", err)
		}
		dto := backupRecordDTO(snapshot)
		result.SafetySnapshot = &BackupSafetySnapshot{Record: &dto}
	} else {
		result.Warnings = append(result.Warnings, "no_live_db: current database does not exist; restore will create it without a safety snapshot")
	}

	if err := s.restore.Replace(ctx, record, targetDBPath); err != nil {
		result.Warnings = append(result.Warnings, "rollback: restore failed after safety checks; live database should remain unchanged and any safety snapshot was preserved")
		return result, fmt.Errorf("replace target database: %w", err)
	}
	result.Replaced = true
	return result, nil
}

func backupRecordDTO(record BackupRecord) BackupRecordDTO {
	return BackupRecordDTO{
		ID:            record.ID,
		File:          record.File,
		SidecarFile:   record.SidecarFile,
		CreatedAt:     record.CreatedAt,
		SchemaVersion: record.SchemaVersion,
		SizeBytes:     record.SizeBytes,
		SHA256:        record.SHA256,
		Metadata:      record.Metadata,
	}
}

func sensitiveBackupWarnings() []string {
	return []string{"Backups contain sensitive billing data; protect them like the live database."}
}

func validateRestoreSelector(req BackupRestoreRequest) error {
	hasID := strings.TrimSpace(req.BackupID) != ""
	hasFile := strings.TrimSpace(req.File) != ""
	if hasID == hasFile {
		return fmt.Errorf("exactly_one_selector: provide exactly one of --id or --file")
	}
	return nil
}

func requireRestoreConfirmation(req BackupRestoreRequest, record BackupRecord) error {
	if req.Force {
		return nil
	}
	token := strings.TrimSpace(req.Confirm)
	if token == "" {
		return fmt.Errorf("missing_confirmation: rerun with --confirm %s or --force after stopping other Billar processes", restoreConfirmToken(record))
	}
	if token != record.ID && token != filepath.Base(record.File) {
		return fmt.Errorf("confirmation_mismatch: --confirm must match backup id %q or backup file basename %q", record.ID, filepath.Base(record.File))
	}
	return nil
}

func requireSourceTargetMatchUnlessForced(req BackupRestoreRequest, record BackupRecord, targetDBPath string) error {
	if req.Force || strings.TrimSpace(record.SourceDBPath) == "" || strings.TrimSpace(targetDBPath) == "" {
		return nil
	}
	if filepath.Clean(record.SourceDBPath) != filepath.Clean(targetDBPath) {
		return fmt.Errorf("source_target_mismatch: backup was created from %q but target database is %q; rerun with --force to acknowledge", record.SourceDBPath, targetDBPath)
	}
	return nil
}

func restoreConfirmToken(record BackupRecord) string {
	if record.ID != "" {
		return record.ID
	}
	return filepath.Base(record.File)
}

func concurrentRestoreWarning() string {
	return "concurrent_processes: stop other Billar processes and external SQLite clients before restoring"
}

func restorePathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
