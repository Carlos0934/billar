package app

type SetupReportDTO struct {
	Project        string   `json:"project" toon:"project"`
	DBPath         string   `json:"db_path" toon:"db_path"`
	ExportDir      string   `json:"export_dir" toon:"export_dir"`
	BackupDir      string   `json:"backup_dir" toon:"backup_dir"`
	Created        []string `json:"created" toon:"created"`
	AlreadyExisted []string `json:"already_existed" toon:"already_existed"`
	NextSteps      []string `json:"next_steps" toon:"next_steps"`
	Warnings       []string `json:"warnings" toon:"warnings"`
}
