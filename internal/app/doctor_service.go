package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type DoctorStore interface {
	Ping(ctx context.Context) error
	SchemaVersion(ctx context.Context) (int, error)
}

type DoctorDBProbe interface {
	Probe(ctx context.Context, dbPath string) (schemaVersion int, err error)
}

type DoctorService struct {
	store  DoctorStore
	config DoctorConfig
}

func NewDoctorService(store DoctorStore, config DoctorConfig) DoctorService {
	return DoctorService{store: store, config: config}
}

func (s DoctorService) Report(ctx context.Context) (DoctorReportDTO, error) {
	report := DoctorReportDTO{
		Project:         strings.TrimSpace(s.config.Project),
		DBPath:          strings.TrimSpace(s.config.DBPath),
		DBPathSource:    doctorPathSource(s.config.DBPathSource),
		ExportDir:       strings.TrimSpace(s.config.ExportDir),
		ExportDirSource: doctorPathSource(s.config.ExportDirSource),
		BackupDir:       strings.TrimSpace(s.config.BackupDir),
		BackupDirSource: doctorPathSource(s.config.BackupDirSource),
		PDFAvailable:    s.config.PDFAvailable,
	}
	report.DBParentDir = filepath.Dir(report.DBPath)
	report.DBParentDirSource = report.DBPathSource
	report.DBPathExists = pathExists(report.DBPath)

	if s.store == nil {
		s.probeDBReadiness(ctx, &report)
	} else if err := s.store.Ping(ctx); err != nil {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db", false, err.Error()))
	} else {
		report.DBReachable = true
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db", true, "reachable"))
	}

	if s.store != nil {
		if version, err := s.store.SchemaVersion(ctx); err == nil {
			report.SchemaVersion = version
		}
	}

	report.DBParentDirExists, report.DBParentDirWritable = checkDirWritable(report.DBParentDir)
	if !report.DBParentDirExists {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db_parent_dir", false, "database parent directory does not exist"))
	} else if !report.DBParentDirWritable {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db_parent_dir", false, "database parent directory is not writable"))
	} else {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db_parent_dir", true, "writable"))
	}

	report.ExportDirSet = strings.TrimSpace(report.ExportDir) != ""
	report.ExportDirExists, report.ExportDirWritable = checkDirWritable(report.ExportDir)
	if !report.ExportDirSet {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("export_dir", false, "BILLAR_EXPORT_DIR is not configured"))
	} else if !report.ExportDirExists {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("export_dir", false, "export directory does not exist"))
	} else if !report.ExportDirWritable {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("export_dir", false, "export directory is not writable"))
	} else {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("export_dir", true, "writable"))
	}

	report.BackupDirExists, report.BackupDirWritable = checkDirWritable(report.BackupDir)
	if !report.BackupDirExists {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("backup_dir", false, "backup directory does not exist"))
	} else if !report.BackupDirWritable {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("backup_dir", false, "backup directory is not writable"))
	} else {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("backup_dir", true, "writable"))
	}

	if !report.DBParentDirExists || !report.ExportDirExists || !report.BackupDirExists {
		report.NextSteps = append(report.NextSteps, "billar setup")
	}

	if report.PDFAvailable {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("pdf", true, "available"))
	} else {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("pdf", false, "not configured"))
	}

	return report, nil
}

func (s DoctorService) probeDBReadiness(ctx context.Context, report *DoctorReportDTO) {
	if report == nil {
		return
	}
	if s.config.DBProbe == nil || !report.DBPathExists {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db", false, "database not opened; readiness checked without creating or migrating SQLite"))
		return
	}
	version, err := s.config.DBProbe.Probe(ctx, report.DBPath)
	if err != nil {
		report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db", false, err.Error()))
		return
	}
	report.DBReachable = true
	report.SchemaVersion = version
	report.CommandHealth = append(report.CommandHealth, doctorCommandHealth("db", true, "reachable via read-only probe"))
}

func doctorCommandHealth(name string, ok bool, message string) DoctorCommandHealthDTO {
	status := "ok"
	if !ok {
		status = "fail"
	}
	return DoctorCommandHealthDTO{Name: name, Status: status, Message: message}
}

func checkDirWritable(dir string) (bool, bool) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false, false
	}
	info, err := os.Stat(dir)
	if err != nil {
		return false, false
	}
	if !info.IsDir() {
		return true, false
	}
	testFile := filepath.Join(dir, ".billar-doctor-write-test")
	file, err := os.OpenFile(testFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return true, false
	}
	if closeErr := file.Close(); closeErr != nil {
		_ = os.Remove(testFile)
		return true, false
	}
	if err := os.Remove(testFile); err != nil {
		return true, false
	}
	return true, true
}

func pathExists(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func doctorPathSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "default"
	}
	return source
}

func formatDoctorSchemaVersion(version int) string {
	if version <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d", version)
}
