package app

type RuntimePath struct {
	Path   string `json:"path" toon:"path"`
	Source string `json:"source" toon:"source"`
}

type RuntimePaths struct {
	DBPath    RuntimePath `json:"db_path" toon:"db_path"`
	ExportDir RuntimePath `json:"export_dir" toon:"export_dir"`
	BackupDir RuntimePath `json:"backup_dir" toon:"backup_dir"`
}
