# Invoice operations

This guide covers invoice import, inspection, metadata corrections, and PDF export through the Billar CLI.

## CLI-first policy

Routine billing operations must use the CLI and shared application services. Direct SQLite edits are emergency repair-only and require explicit operator approval before touching the database.

## Import issued invoices

Billar accepts historical issued invoices using the `billar.invoice.import/v1` JSON payload:

```bash
billar invoice import --file examples/imports/INV-2026-00001.json
billar invoice import --stdin < examples/imports/INV-2026-00001.json
```

The import flow expects existing issuer/customer references or customer resolution by existing tax ID/legal name. It does not parse PDFs or create profiles automatically. Imported line `amount_minor` is authoritative; `quantity_display` and `unit_price_display` are preserved for display and PDF rendering.

## Review invoices

```bash
billar invoice show --id inv_123
billar invoice inspect --id inv_123 --format json
billar invoice list --customer-id cus_123 --status issued
```

Use `invoice inspect` for the complete DTO, including imported identity fields and line details.

## Correct metadata safely

```bash
billar invoice update-metadata --id inv_123 --invoice-date 2026-05-01 --due-date 2026-05-15 --payment-terms "Net 15" --payment-communication "Use invoice number INV-001"
```

`invoice update-metadata` changes non-financial metadata only: `invoice_date`, `due_date`, `payment_terms`, and `payment_communication`. Omitted or empty flags leave existing values unchanged. Imported identity fields and financial totals remain unchanged.

## Render PDFs

```bash
billar invoice pdf inv_123 --out ./exports/invoice.pdf --format json
billar invoice pdf inv_123 --format text
```

When `--out` is omitted, `BILLAR_EXPORT_DIR` is the export root for the default invoice filename. If `BILLAR_EXPORT_DIR` is unset, pass `--out <path>` explicitly. When unset in config, `BILLAR_EXPORT_DIR` defaults under the DB parent directory; `.env.example` shows the optional override.

PDF rendering returns file metadata in `text`, `json`, or `toon` formats.
