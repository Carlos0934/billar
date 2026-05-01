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
