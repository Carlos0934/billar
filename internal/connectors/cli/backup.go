package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

func (c Command) runBackup(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: billar backup <create|list> [flags]")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		_, err := io.WriteString(out, "usage: billar backup <create|list> [flags]\n")
		return err
	}
	if c.backup == nil {
		return errors.New("backup service is required")
	}

	subcommand := strings.ToLower(args[0])
	switch subcommand {
	case "create":
		format, err := parseFormatFlag("backup create", args[1:])
		if err != nil {
			return err
		}
		result, err := c.backup.Create(ctx)
		if err != nil {
			return fmt.Errorf("run backup create command: %w", err)
		}
		output := OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeBackupCreateText(w, result, c.colorEnabled) }}
		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write backup create output: %w", err)
		}
		return nil
	case "list":
		format, err := parseFormatFlag("backup list", args[1:])
		if err != nil {
			return err
		}
		result, err := c.backup.List(ctx)
		if err != nil {
			return fmt.Errorf("run backup list command: %w", err)
		}
		output := OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeBackupListText(w, result, c.colorEnabled) }}
		if err := WriteOutput(out, format, output); err != nil {
			return fmt.Errorf("write backup list output: %w", err)
		}
		return nil
	case "restore":
		return errors.New("backup restore is deferred to a future change; see README restore notes for the manual file-copy workaround")
	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"backup", args[0]}, " "))
	}
}

func writeBackupCreateText(out io.Writer, record app.BackupRecordDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Backup Created").Divider("─────────────────────")
	writeBackupRecordFields(view, record)
	for _, warning := range record.Warnings {
		view.Line("Warning: " + warning)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeBackupListText(out io.Writer, list app.BackupListDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Backups").Divider("──────────────")
	view.Field("dir", list.Dir)
	if len(list.Backups) == 0 {
		view.Line("No backups found.")
	}
	for i, record := range list.Backups {
		prefix := fmt.Sprintf("backups.%d.", i)
		view.Field(prefix+"id", record.ID)
		view.Field(prefix+"file", record.File)
		view.Field(prefix+"metadata", fmt.Sprintf("%t", record.Metadata))
		if record.SHA256 != "" {
			view.Field(prefix+"sha256", record.SHA256)
		}
		if record.SizeBytes > 0 {
			view.Field(prefix+"size_bytes", fmt.Sprintf("%d", record.SizeBytes))
		}
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func writeBackupRecordFields(view *textView, record app.BackupRecordDTO) {
	view.Field("id", record.ID)
	view.Field("file", record.File)
	view.Field("sidecar_file", record.SidecarFile)
	view.Field("created_at", record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"))
	view.Field("schema_version", fmt.Sprintf("%d", record.SchemaVersion))
	view.Field("size_bytes", fmt.Sprintf("%d", record.SizeBytes))
	view.Field("sha256", record.SHA256)
	view.Field("metadata", fmt.Sprintf("%t", record.Metadata))
}
