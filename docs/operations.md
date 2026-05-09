# Operations runbook

Use this runbook for local setup, configuration checks, and SQLite backup/restore.

## Setup and doctor

```bash
billar setup
billar doctor --format json
billar health
```

`billar setup` creates the resolved DB parent, export directory, and backup directory. It does not write `.env`. `billar doctor` is read-only and reports path readiness, source (`configured` or `default`), and actionable setup hints.

## Configuration defaults

Billar loads `.env` from the current working directory only. Existing non-empty environment variables win over `.env` values.

- `BILLAR_DB_PATH`: optional override for the live SQLite database. When unset, Billar resolves a per-user `billar.db` using `XDG_DATA_HOME`, then `os.UserConfigDir()`, then `~/.local/share/billar/billar.db`.
- `BILLAR_EXPORT_DIR`: optional override for CLI PDF/file exports. When unset, it defaults to `<db-parent>/exports`.
- `BILLAR_BACKUP_DIR`: optional override for local backups. When unset, it defaults to `<db-parent>/backups`.

## Create and list backups

```bash
billar backup create --format json
billar backup list --format text
```

`billar backup create` writes a `.db` SQLite snapshot plus a `.db.json` sidecar. The sidecar records `created_at`, `source_db_path`, `size_bytes`, `sha256`, and `schema_version`. Backup data is sensitive and unencrypted.

## Restore prerequisites

Before running `billar backup restore`:

1. Stop other Billar processes and external SQLite clients. Restore output includes a `concurrent_processes` warning.
2. Confirm the target DB path from `billar doctor`; this is the resolved `BILLAR_DB_PATH`.
3. Choose exactly one source: `--id <backup-id>` from `BILLAR_BACKUP_DIR`, or `--file <path-to-backup.db>`.
4. Run a dry run first.

```bash
billar backup restore --id billar-20260501T120000Z-schema4 --dry-run --format json
```

## Destructive restore

```bash
billar backup restore --id billar-20260501T120000Z-schema4 --confirm billar-20260501T120000Z-schema4
billar backup restore --file /path/to/billar-20260501T120000Z-schema4.db --confirm billar-20260501T120000Z-schema4.db
billar backup restore --id billar-20260501T120000Z-schema4 --force
```

`--confirm <token>` must match the backup ID or file basename. `--force` is for automation and does not bypass validation. Validation checks sidecar metadata, size, sha256, SQLite integrity, schema version, and required Billar tables.

When a live DB exists, restore first creates a pre-restore safety snapshot in `BILLAR_BACKUP_DIR` and reports it. If no live DB exists, restore reports `safety_snapshot: null`.

## Post-restore verification

```bash
billar doctor --format json
billar health
```

Then inspect representative customers and invoices before resuming billing work.

## Exit codes

- `0`: success or dry-run plan printed.
- `2`: usage or confirmation error.
- `3`: validation failure.
- `4`: snapshot, copy, or rename runtime failure.
- `5`: unexpected internal error.
