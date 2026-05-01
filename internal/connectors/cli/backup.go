package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

func (c Command) runBackup(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: billar backup <create|list|restore> [flags]")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		_, err := io.WriteString(out, "usage: billar backup <create|list|restore> [flags]\n")
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
		return c.runBackupRestore(ctx, args[1:], out)
	default:
		return fmt.Errorf("unknown command %q", strings.Join([]string{"backup", args[0]}, " "))
	}
}

func (c Command) runBackupRestore(ctx context.Context, args []string, out io.Writer) error {
	req, format, err := parseBackupRestoreFlags(args)
	if err != nil {
		if format != "" {
			_ = writeBackupRestoreErrorOutput(out, format, app.BackupRestoreResultDTO{DryRun: req.DryRun, Warnings: destructiveRestoreWarnings(req)}, err, c.colorEnabled)
		}
		return exitError{code: 2, err: err}
	}
	result, err := c.backup.Restore(ctx, req)
	if err != nil {
		if !req.DryRun && !hasWarning(result.Warnings, "concurrent_processes") {
			result.Warnings = append([]string{concurrentRestoreWarningText()}, result.Warnings...)
		}
		wrapped := fmt.Errorf("run backup restore command: %w", err)
		if writeErr := writeBackupRestoreErrorOutput(out, format, result, wrapped, c.colorEnabled); writeErr != nil {
			return exitError{code: 5, err: fmt.Errorf("write backup restore error output: %w", writeErr)}
		}
		return exitError{code: classifyRestoreError(err), err: wrapped}
	}
	output := OutputResult{Payload: result, TextWriter: func(w io.Writer) error { return writeBackupRestoreText(w, result, c.colorEnabled) }}
	if err := WriteOutput(out, format, output); err != nil {
		return exitError{code: 5, err: fmt.Errorf("write backup restore output: %w", err)}
	}
	return nil
}

func parseBackupRestoreFlags(args []string) (app.BackupRestoreRequest, Format, error) {
	flags := flag.NewFlagSet("backup restore", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var req app.BackupRestoreRequest
	var formatValue string
	flags.StringVar(&req.BackupID, "id", "", "backup id to restore")
	flags.StringVar(&req.File, "file", "", "backup database file to restore")
	flags.BoolVar(&req.DryRun, "dry-run", false, "validate and print restore plan without replacing the database")
	flags.StringVar(&req.Confirm, "confirm", "", "confirmation token matching backup id or file basename")
	flags.BoolVar(&req.Force, "force", false, "bypass confirmation and source-target mismatch warnings")
	flags.StringVar(&formatValue, "format", string(FormatText), "output format")
	if err := flags.Parse(args); err != nil {
		return app.BackupRestoreRequest{}, "", fmt.Errorf("usage: billar backup restore (--id <id> | --file <path>) [--dry-run] [--confirm <token>] [--force] [--format text|json|toon]: %w", err)
	}
	format, err := ParseFormat(formatValue)
	if err != nil {
		return app.BackupRestoreRequest{}, "", err
	}
	if flags.NArg() != 0 {
		return req, format, errors.New("usage: billar backup restore (--id <id> | --file <path>) [--dry-run] [--confirm <token>] [--force] [--format text|json|toon]")
	}
	if (strings.TrimSpace(req.BackupID) == "") == (strings.TrimSpace(req.File) == "") {
		return req, format, errors.New("exactly_one_selector: exactly one selector is required: provide --id or --file")
	}
	return req, format, nil
}

type backupRestoreErrorOutput struct {
	Restore  *app.BackupRestoreResultDTO `json:"restore,omitempty" toon:"restore,omitempty"`
	Error    backupRestoreErrorDTO       `json:"error" toon:"error"`
	Warnings []string                    `json:"warnings,omitempty" toon:"warnings,omitempty"`
}

type backupRestoreErrorDTO struct {
	Code    string `json:"code" toon:"code"`
	Message string `json:"message" toon:"message"`
}

func writeBackupRestoreErrorOutput(out io.Writer, format Format, result app.BackupRestoreResultDTO, err error, colorEnabled bool) error {
	payload := backupRestoreErrorOutput{
		Error:    backupRestoreErrorDTO{Code: restoreErrorCode(err), Message: err.Error()},
		Warnings: append([]string{}, result.Warnings...),
	}
	if result.Backup.ID != "" || result.Backup.File != "" || result.TargetDBPath != "" || result.Validation.BinarySchema != 0 || result.Validation.BackupSchema != 0 {
		copyResult := result
		payload.Restore = &copyResult
	}
	output := OutputResult{Payload: payload, TextWriter: func(w io.Writer) error { return writeBackupRestoreErrorText(w, payload, colorEnabled) }}
	return WriteOutput(out, format, output)
}

func writeBackupRestoreErrorText(out io.Writer, result backupRestoreErrorOutput, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Backup Restore Failed").Divider("────────────────────────────")
	view.Field("error.code", result.Error.Code)
	view.Field("error.message", result.Error.Message)
	if result.Restore != nil {
		view.Field("backup.id", result.Restore.Backup.ID)
		view.Field("backup.file", result.Restore.Backup.File)
		view.Field("target_db_path", result.Restore.TargetDBPath)
		view.Field("validation.ok", fmt.Sprintf("%t", result.Restore.Validation.OK))
	}
	for _, warning := range result.Warnings {
		view.Line("Warning: " + warning)
	}
	_, err := io.WriteString(out, view.Build())
	return err
}

func destructiveRestoreWarnings(req app.BackupRestoreRequest) []string {
	if req.DryRun {
		return nil
	}
	return []string{concurrentRestoreWarningText()}
}

func hasWarning(warnings []string, token string) bool {
	for _, warning := range warnings {
		if strings.Contains(warning, token) {
			return true
		}
	}
	return false
}

func concurrentRestoreWarningText() string {
	return "concurrent_processes: stop other Billar processes and external SQLite clients before restoring"
}

func restoreErrorCode(err error) string {
	message := strings.ToLower(err.Error())
	for _, token := range []string{
		"exactly_one_selector", "missing_confirmation", "confirmation_mismatch", "source_target_mismatch",
		"id_basename_mismatch", "sidecar_missing", "sidecar_parse_error", "hash_mismatch", "size_mismatch", "integrity_check_failed", "schema_newer_than_binary", "schema_sidecar_mismatch", "schema_migrations_missing", "missing_required_tables", "source_is_target", "temp_hash_mismatch",
	} {
		if strings.Contains(message, token) {
			return token
		}
	}
	if strings.Contains(message, "usage:") {
		return "usage_error"
	}
	return "internal_error"
}

type exitError struct {
	code int
	err  error
}

func (e exitError) Error() string { return e.err.Error() }
func (e exitError) Unwrap() error { return e.err }
func (e exitError) ExitCode() int { return e.code }

func commandExitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return 1
}

func ExitCode(err error) int {
	return commandExitCode(err)
}

func classifyRestoreError(err error) int {
	message := strings.ToLower(err.Error())
	for _, token := range []string{"missing_confirmation", "confirmation_mismatch", "source_target_mismatch", "exactly_one_selector"} {
		if strings.Contains(message, token) {
			return 2
		}
	}
	for _, token := range []string{"validate backup", "sidecar", "hash_mismatch", "size_mismatch", "integrity", "schema", "missing_required_tables", "source_is_target"} {
		if strings.Contains(message, token) {
			return 3
		}
	}
	for _, token := range []string{"create safety snapshot", "replace target database", "snapshot", "rename", "copy"} {
		if strings.Contains(message, token) {
			return 4
		}
	}
	return 5
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

func writeBackupRestoreText(out io.Writer, result app.BackupRestoreResultDTO, colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Backup Restore").Divider("─────────────────────")
	view.Field("backup.id", result.Backup.ID)
	view.Field("backup.file", result.Backup.File)
	view.Field("target_db_path", result.TargetDBPath)
	view.Field("dry_run", fmt.Sprintf("%t", result.DryRun))
	view.Field("replaced", fmt.Sprintf("%t", result.Replaced))
	view.Field("validation.ok", fmt.Sprintf("%t", result.Validation.OK))
	view.Field("validation.backup_schema", fmt.Sprintf("%d", result.Validation.BackupSchema))
	view.Field("validation.binary_schema", fmt.Sprintf("%d", result.Validation.BinarySchema))
	if result.SafetySnapshot != nil {
		if result.SafetySnapshot.Record != nil {
			view.Field("safety_snapshot.id", result.SafetySnapshot.Record.ID)
			view.Field("safety_snapshot.file", result.SafetySnapshot.Record.File)
		} else if result.SafetySnapshot.Skipped {
			view.Field("safety_snapshot.skipped", "true")
			view.Field("safety_snapshot.reason", result.SafetySnapshot.Reason)
		}
	}
	for _, warning := range result.Warnings {
		view.Line("Warning: " + warning)
	}
	if strings.TrimSpace(result.Backup.File) != "" {
		view.Field("confirm_token", filepath.Base(result.Backup.File))
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
