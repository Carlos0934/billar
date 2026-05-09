package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
	connectorcli "github.com/Carlos0934/billar/internal/connectors/cli"
	"github.com/Carlos0934/billar/internal/infra/backup"
	"github.com/Carlos0934/billar/internal/infra/config"
	"github.com/Carlos0934/billar/internal/infra/exportfs"
	"github.com/Carlos0934/billar/internal/infra/pdf"
	infrasqlite "github.com/Carlos0934/billar/internal/infra/sqlite"
)

func newCommand(cfg config.Config, store *infrasqlite.Store) connectorcli.Command {
	legalEntityStore := infrasqlite.NewLegalEntityStore(store)
	issuerProfileStore := infrasqlite.NewIssuerProfileStore(store)
	customerProfileStore := infrasqlite.NewCustomerProfileStore(store)
	agreementStore := infrasqlite.NewServiceAgreementStore(store)
	timeEntryStore := infrasqlite.NewTimeEntryStore(store)
	invoiceStore := infrasqlite.NewInvoiceStore(store)
	quoteStore := infrasqlite.NewQuoteStore(store)
	invoiceSequenceStore := infrasqlite.NewInvoiceSequenceStore(store)
	invoiceService := app.NewInvoiceService(invoiceStore, timeEntryStore, agreementStore, customerProfileStore, invoiceSequenceStore, issuerProfileStore, legalEntityStore)
	invoicePDFService := app.NewInvoicePDFService(invoiceStore, timeEntryStore, customerProfileStore, issuerProfileStore, legalEntityStore, pdf.Renderer{}, exportfs.FlexibleWriter{Root: cfg.ExportDir})
	invoiceProvider := app.NewInvoiceProvider(invoiceService, invoicePDFService)
	runtimePaths := runtimePathsFromConfig(cfg)
	doctorService := app.NewDoctorService(invoiceStore, app.DoctorConfig{Project: cfg.AppName, DBPath: cfg.DBPath, DBPathSource: cfg.DBPathSource, ExportDir: cfg.ExportDir, ExportDirSource: cfg.ExportDirSource, BackupDir: cfg.BackupDir, BackupDirSource: cfg.BackupDirSource, PDFAvailable: true})
	setupService := app.NewSetupService(cfg.AppName, runtimePaths)
	backupService := newBackupService(runtimePaths)

	quoteService := app.NewQuoteService(quoteStore, customerProfileStore, agreementStore)

	return connectorcli.NewCommand(
		app.NewHealthService(cfg.AppName),
		app.NewLegalEntityService(legalEntityStore),
		app.NewIssuerProfileService(legalEntityStore, issuerProfileStore),
		app.NewCustomerProfileService(legalEntityStore, customerProfileStore),
		app.NewAgreementService(agreementStore, customerProfileStore),
		app.NewTimeEntryService(timeEntryStore, customerProfileStore, agreementStore),
		invoiceProvider,
		cfg.ColorEnabled,
		doctorService,
	).WithExportDir(cfg.ExportDir).WithSetupService(setupService).WithBackupService(backupService).WithQuoteService(quoteService)
}

func newPreStoreCommand(cfg config.Config) connectorcli.Command {
	runtimePaths := runtimePathsFromConfig(cfg)
	doctorService := app.NewDoctorService(nil, app.DoctorConfig{Project: cfg.AppName, DBPath: cfg.DBPath, DBPathSource: cfg.DBPathSource, DBProbe: infrasqlite.DoctorReadOnlyProbe{}, ExportDir: cfg.ExportDir, ExportDirSource: cfg.ExportDirSource, BackupDir: cfg.BackupDir, BackupDirSource: cfg.BackupDirSource, PDFAvailable: true})
	setupService := app.NewSetupService(cfg.AppName, runtimePaths)
	backupService := newBackupService(runtimePaths)
	return connectorcli.NewCommand(app.NewHealthService(cfg.AppName), nil, nil, nil, nil, nil, nil, cfg.ColorEnabled, doctorService).WithSetupService(setupService).WithBackupService(backupService)
}

func newBackupService(runtimePaths app.RuntimePaths) app.BackupService {
	return app.NewBackupServiceWithRestore(
		backupSnapshotterAdapter{inner: backup.Snapshotter{}},
		backupListerAdapter{inner: backup.Lister{}},
		staticRuntimePathsProvider{paths: runtimePaths},
		backupRestorerAdapter{inner: backup.Restorer{}},
		latestMigrationSchemaProvider{},
	)
}

func runtimePathsFromConfig(cfg config.Config) app.RuntimePaths {
	return app.RuntimePaths{
		DBPath:    app.RuntimePath{Path: cfg.DBPath, Source: cfg.DBPathSource},
		ExportDir: app.RuntimePath{Path: cfg.ExportDir, Source: cfg.ExportDirSource},
		BackupDir: app.RuntimePath{Path: cfg.BackupDir, Source: cfg.BackupDirSource},
	}
}

type staticRuntimePathsProvider struct {
	paths app.RuntimePaths
}

func (p staticRuntimePathsProvider) RuntimePaths(context.Context) (app.RuntimePaths, error) {
	return p.paths, nil
}

type backupSnapshotterAdapter struct {
	inner backup.Snapshotter
}

func (a backupSnapshotterAdapter) Create(ctx context.Context, dbPath, destDir string) (app.BackupRecord, error) {
	record, err := a.inner.Create(ctx, dbPath, destDir)
	if err != nil {
		return app.BackupRecord{}, err
	}
	return backupRecordToApp(record), nil
}

type backupListerAdapter struct {
	inner backup.Lister
}

func (a backupListerAdapter) List(ctx context.Context, dir string) ([]app.BackupRecord, error) {
	records, err := a.inner.List(ctx, dir)
	if err != nil {
		return nil, err
	}
	mapped := make([]app.BackupRecord, 0, len(records))
	for _, record := range records {
		mapped = append(mapped, backupRecordToApp(record))
	}
	return mapped, nil
}

func backupRecordToApp(record backup.Record) app.BackupRecord {
	return app.BackupRecord{
		ID:            record.ID,
		File:          record.File,
		SidecarFile:   record.SidecarFile,
		CreatedAt:     record.CreatedAt,
		SchemaVersion: record.SchemaVersion,
		SizeBytes:     record.SizeBytes,
		SHA256:        record.SHA256,
		SourceDBPath:  record.SourceDBPath,
		Metadata:      record.Metadata,
	}
}

type backupRestorerAdapter struct {
	inner backup.Restorer
}

func (a backupRestorerAdapter) Resolve(ctx context.Context, req app.BackupRestoreRequest, backupDir string) (app.BackupRecord, error) {
	record, err := a.inner.Resolve(ctx, backup.RestoreRequest{BackupID: req.BackupID, File: req.File}, backupDir)
	if err != nil {
		return app.BackupRecord{}, err
	}
	return backupRecordToApp(record), nil
}

func (a backupRestorerAdapter) Validate(ctx context.Context, record app.BackupRecord, targetDBPath string, binarySchema int) (app.BackupValidation, error) {
	validation, err := a.inner.Validate(ctx, appRecordToBackup(record), targetDBPath, binarySchema)
	return app.BackupValidation{
		OK:           validation.OK,
		SidecarOK:    validation.SidecarOK,
		HashOK:       validation.HashOK,
		SizeOK:       validation.SizeOK,
		IntegrityOK:  validation.IntegrityOK,
		TablesOK:     validation.TablesOK,
		SchemaOK:     validation.SchemaOK,
		BackupSchema: validation.BackupSchema,
		BinarySchema: validation.BinarySchema,
	}, err
}

func (a backupRestorerAdapter) Replace(ctx context.Context, record app.BackupRecord, targetDBPath string) error {
	return a.inner.Replace(ctx, appRecordToBackup(record), targetDBPath)
}

func appRecordToBackup(record app.BackupRecord) backup.Record {
	return backup.Record{
		ID:            record.ID,
		File:          record.File,
		SidecarFile:   record.SidecarFile,
		CreatedAt:     record.CreatedAt,
		SchemaVersion: record.SchemaVersion,
		SizeBytes:     record.SizeBytes,
		SHA256:        record.SHA256,
		SourceDBPath:  record.SourceDBPath,
		Metadata:      record.Metadata,
	}
}

type latestMigrationSchemaProvider struct{}

func (latestMigrationSchemaProvider) LatestSchemaVersion() int {
	version, err := infrasqlite.LatestMigrationVersion()
	if err != nil {
		return 0
	}
	return version
}

func main() {
	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !commandNeedsStore(os.Args[1:]) {
		cmd := newPreStoreCommand(cfg)
		if err := cmd.Run(ctx, os.Args[1:], os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(connectorcli.ExitCode(err))
		}
		return
	}
	store, err := openConfiguredStore(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	cmd := newCommand(cfg, store)

	if err := cmd.Run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(connectorcli.ExitCode(err))
	}
}

func commandNeedsStore(args []string) bool {
	if len(args) == 0 {
		return false
	}
	subcommand := strings.ToLower(strings.TrimSpace(args[0]))
	switch subcommand {
	case "--help", "-h", "help", "setup", "doctor", "backup":
		return false
	default:
		return true
	}
}

func openConfiguredStore(cfg config.Config) (*infrasqlite.Store, error) {
	store, err := infrasqlite.Open(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database at %q: %w; set BILLAR_DB_PATH to choose a writable database path", cfg.DBPath, err)
	}
	return store, nil
}
