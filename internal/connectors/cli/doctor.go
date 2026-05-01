package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

func (c Command) runDoctor(ctx context.Context, args []string, out io.Writer) error {
	if c.doctor == nil {
		return errors.New("doctor service is required")
	}
	format, err := parseFormatFlag("doctor", args)
	if err != nil {
		return err
	}
	report, err := c.doctor.Report(ctx)
	if err != nil {
		return fmt.Errorf("run doctor command: %w", err)
	}
	output := OutputResult{Payload: report, TextWriter: func(w io.Writer) error { return writeDoctorText(w, report, c.colorEnabled) }}
	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write doctor output: %w", err)
	}
	if doctorReportHasFailedReadiness(report) {
		return errors.New("doctor readiness failed")
	}
	return nil
}

func doctorReportHasFailedReadiness(report app.DoctorReportDTO) bool {
	for _, check := range report.CommandHealth {
		if strings.EqualFold(strings.TrimSpace(check.Status), "fail") {
			return true
		}
	}
	return false
}

func writeDoctorText(out io.Writer, report app.DoctorReportDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Doctor").Divider("─────────────")
	view.Field("project", report.Project)
	view.Field("db_path", report.DBPath)
	view.Field("db_path_source", report.DBPathSource)
	view.Field("db_parent_dir", report.DBParentDir)
	view.Field("db_parent_dir_source", report.DBParentDirSource)
	view.Field("db_parent_dir_exists", fmt.Sprintf("%t", report.DBParentDirExists))
	view.Field("db_parent_dir_writable", fmt.Sprintf("%t", report.DBParentDirWritable))
	view.Field("schema_version", formatDoctorSchemaVersionForText(report.SchemaVersion))
	view.Field("db_reachable", fmt.Sprintf("%t", report.DBReachable))
	view.Field("export_dir", report.ExportDir)
	view.Field("export_dir_source", report.ExportDirSource)
	view.Field("export_dir_exists", fmt.Sprintf("%t", report.ExportDirExists))
	view.Field("export_dir_set", fmt.Sprintf("%t", report.ExportDirSet))
	view.Field("export_dir_writable", fmt.Sprintf("%t", report.ExportDirWritable))
	view.Field("backup_dir", report.BackupDir)
	view.Field("backup_dir_source", report.BackupDirSource)
	view.Field("backup_dir_exists", fmt.Sprintf("%t", report.BackupDirExists))
	view.Field("backup_dir_writable", fmt.Sprintf("%t", report.BackupDirWritable))
	view.Field("pdf_available", fmt.Sprintf("%t", report.PDFAvailable))
	if !report.ExportDirSet {
		view.Line("Guidance: set BILLAR_EXPORT_DIR or pass --out when exporting PDFs.")
	}
	for _, check := range report.CommandHealth {
		view.Field("command_health."+check.Name, check.Status+" — "+check.Message)
	}
	if len(report.NextSteps) > 0 {
		view.Field("next_steps", strings.Join(report.NextSteps, ", "))
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func formatDoctorSchemaVersionForText(version int) string {
	if version <= 0 {
		return "unknown"
	}
	return fmt.Sprintf("%d", version)
}
