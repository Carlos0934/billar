package app

import (
	"context"
	"fmt"
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

type BackupService struct {
	snap  BackupSnapshotter
	list  BackupLister
	paths BackupPathsProvider
}

func NewBackupService(snap BackupSnapshotter, list BackupLister, paths BackupPathsProvider) BackupService {
	return BackupService{snap: snap, list: list, paths: paths}
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
