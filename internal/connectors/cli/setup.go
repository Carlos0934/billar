package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

func (c Command) runSetup(ctx context.Context, args []string, out io.Writer) error {
	if c.setup == nil {
		return errors.New("setup service is required")
	}
	format, err := parseFormatFlag("setup", args)
	if err != nil {
		return err
	}
	report, err := c.setup.Run(ctx)
	if err != nil {
		return fmt.Errorf("run setup command: %w", err)
	}
	output := OutputResult{Payload: report, TextWriter: func(w io.Writer) error { return writeSetupText(w, report, c.colorEnabled) }}
	if err := WriteOutput(out, format, output); err != nil {
		return fmt.Errorf("write setup output: %w", err)
	}
	return nil
}

func writeSetupText(out io.Writer, report app.SetupReportDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Setup").Divider("────────────")
	view.Field("project", report.Project)
	view.Field("db_path", report.DBPath)
	view.Field("export_dir", report.ExportDir)
	view.Field("backup_dir", report.BackupDir)
	if len(report.Created) > 0 {
		view.Field("created", strings.Join(report.Created, ", "))
	}
	if len(report.AlreadyExisted) > 0 {
		view.Field("already_existed", strings.Join(report.AlreadyExisted, ", "))
	}
	for _, warning := range report.Warnings {
		view.Line("Warning: " + warning)
	}
	if len(report.NextSteps) > 0 {
		view.Field("next_steps", strings.Join(report.NextSteps, ", "))
	}
	_, err := io.WriteString(out, view.Build())
	return err
}
