package app

type DoctorReportDTO struct {
	Project             string                   `json:"project" toon:"project"`
	DBPath              string                   `json:"db_path" toon:"db_path"`
	DBPathSource        string                   `json:"db_path_source" toon:"db_path_source"`
	DBPathExists        bool                     `json:"db_path_exists" toon:"db_path_exists"`
	DBParentDir         string                   `json:"db_parent_dir" toon:"db_parent_dir"`
	DBParentDirSource   string                   `json:"db_parent_dir_source" toon:"db_parent_dir_source"`
	DBParentDirExists   bool                     `json:"db_parent_dir_exists" toon:"db_parent_dir_exists"`
	DBParentDirWritable bool                     `json:"db_parent_dir_writable" toon:"db_parent_dir_writable"`
	SchemaVersion       int                      `json:"schema_version" toon:"schema_version"`
	DBReachable         bool                     `json:"db_reachable" toon:"db_reachable"`
	ExportDir           string                   `json:"export_dir" toon:"export_dir"`
	ExportDirSource     string                   `json:"export_dir_source" toon:"export_dir_source"`
	ExportDirExists     bool                     `json:"export_dir_exists" toon:"export_dir_exists"`
	ExportDirSet        bool                     `json:"export_dir_set" toon:"export_dir_set"`
	ExportDirWritable   bool                     `json:"export_dir_writable" toon:"export_dir_writable"`
	BackupDir           string                   `json:"backup_dir" toon:"backup_dir"`
	BackupDirSource     string                   `json:"backup_dir_source" toon:"backup_dir_source"`
	BackupDirExists     bool                     `json:"backup_dir_exists" toon:"backup_dir_exists"`
	BackupDirWritable   bool                     `json:"backup_dir_writable" toon:"backup_dir_writable"`
	PDFAvailable        bool                     `json:"pdf_available" toon:"pdf_available"`
	CommandHealth       []DoctorCommandHealthDTO `json:"command_health" toon:"command_health"`
	NextSteps           []string                 `json:"next_steps" toon:"next_steps"`
}

type DoctorCommandHealthDTO struct {
	Name    string `json:"name" toon:"name"`
	Status  string `json:"status" toon:"status"`
	Message string `json:"message" toon:"message"`
}

type DoctorConfig struct {
	Project         string
	DBPath          string
	DBPathSource    string
	DBProbe         DoctorDBProbe
	ExportDir       string
	ExportDirSource string
	BackupDir       string
	BackupDirSource string
	PDFAvailable    bool
}
