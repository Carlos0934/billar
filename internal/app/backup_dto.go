package app

import "time"

type BackupRecord struct {
	ID            string    `json:"id" toon:"id"`
	File          string    `json:"file" toon:"file"`
	SidecarFile   string    `json:"sidecar_file" toon:"sidecar_file"`
	CreatedAt     time.Time `json:"created_at" toon:"created_at"`
	SchemaVersion int       `json:"schema_version" toon:"schema_version"`
	SizeBytes     int64     `json:"size_bytes" toon:"size_bytes"`
	SHA256        string    `json:"sha256" toon:"sha256"`
	SourceDBPath  string    `json:"source_db_path" toon:"source_db_path"`
	Metadata      bool      `json:"metadata" toon:"metadata"`
}

type BackupRecordDTO struct {
	ID            string    `json:"id" toon:"id"`
	File          string    `json:"file" toon:"file"`
	SidecarFile   string    `json:"sidecar_file" toon:"sidecar_file"`
	CreatedAt     time.Time `json:"created_at" toon:"created_at"`
	SchemaVersion int       `json:"schema_version" toon:"schema_version"`
	SizeBytes     int64     `json:"size_bytes" toon:"size_bytes"`
	SHA256        string    `json:"sha256" toon:"sha256"`
	Metadata      bool      `json:"metadata" toon:"metadata"`
	Warnings      []string  `json:"warnings,omitempty" toon:"warnings,omitempty"`
}

type BackupListDTO struct {
	Dir     string            `json:"dir" toon:"dir"`
	Backups []BackupRecordDTO `json:"backups" toon:"backups"`
}

type BackupCreateRequest struct{}

type BackupRestoreRequest struct {
	BackupID string `json:"backup_id,omitempty" toon:"backup_id,omitempty"`
	File     string `json:"file,omitempty" toon:"file,omitempty"`
	DryRun   bool   `json:"dry_run" toon:"dry_run"`
	Confirm  string `json:"confirm,omitempty" toon:"confirm,omitempty"`
	Force    bool   `json:"force" toon:"force"`
}

type BackupValidation struct {
	OK           bool `json:"ok" toon:"ok"`
	SidecarOK    bool `json:"sidecar_ok" toon:"sidecar_ok"`
	HashOK       bool `json:"hash_ok" toon:"hash_ok"`
	SizeOK       bool `json:"size_ok" toon:"size_ok"`
	IntegrityOK  bool `json:"integrity_ok" toon:"integrity_ok"`
	TablesOK     bool `json:"tables_ok" toon:"tables_ok"`
	SchemaOK     bool `json:"schema_ok" toon:"schema_ok"`
	BackupSchema int  `json:"backup_schema" toon:"backup_schema"`
	BinarySchema int  `json:"binary_schema" toon:"binary_schema"`
}

type BackupSafetySnapshot struct {
	Record  *BackupRecordDTO `json:"record,omitempty" toon:"record,omitempty"`
	Skipped bool             `json:"skipped" toon:"skipped"`
	Reason  string           `json:"reason,omitempty" toon:"reason,omitempty"`
}

type BackupRestoreResultDTO struct {
	Backup         BackupRecordDTO       `json:"backup" toon:"backup"`
	TargetDBPath   string                `json:"target_db_path" toon:"target_db_path"`
	DryRun         bool                  `json:"dry_run" toon:"dry_run"`
	Replaced       bool                  `json:"replaced" toon:"replaced"`
	SafetySnapshot *BackupSafetySnapshot `json:"safety_snapshot" toon:"safety_snapshot"`
	Validation     BackupValidation      `json:"validation" toon:"validation"`
	Warnings       []string              `json:"warnings,omitempty" toon:"warnings,omitempty"`
}
