package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SetupService struct {
	project string
	paths   RuntimePaths
}

func NewSetupService(project string, paths RuntimePaths) SetupService {
	return SetupService{project: strings.TrimSpace(project), paths: paths}
}

func (s SetupService) Run(ctx context.Context) (SetupReportDTO, error) {
	select {
	case <-ctx.Done():
		return SetupReportDTO{}, ctx.Err()
	default:
	}

	report := SetupReportDTO{
		Project:   s.project,
		DBPath:    strings.TrimSpace(s.paths.DBPath.Path),
		ExportDir: strings.TrimSpace(s.paths.ExportDir.Path),
		BackupDir: strings.TrimSpace(s.paths.BackupDir.Path),
		NextSteps: []string{
			"billar issuer create",
			"billar customer create",
			"billar doctor",
		},
		Warnings: []string{"Backups are unencrypted local snapshots containing sensitive billing data; protect them like the live database."},
	}

	for _, dir := range []string{filepath.Dir(report.DBPath), report.ExportDir, report.BackupDir} {
		created, err := ensureRuntimeDir(dir)
		if err != nil {
			return SetupReportDTO{}, err
		}
		if created {
			report.Created = append(report.Created, dir)
		} else {
			report.AlreadyExisted = append(report.AlreadyExisted, dir)
		}
	}

	return report, nil
}

func ensureRuntimeDir(dir string) (bool, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" || dir == "." {
		return false, fmt.Errorf("runtime directory path is required")
	}
	if info, err := os.Stat(dir); err == nil {
		if !info.IsDir() {
			return false, fmt.Errorf("runtime path %q exists but is not a directory", dir)
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("check runtime directory %q: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return false, fmt.Errorf("create runtime directory %q: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return false, fmt.Errorf("secure runtime directory %q: %w", dir, err)
	}
	return true, nil
}
