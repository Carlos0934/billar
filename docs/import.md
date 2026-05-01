# Invoice JSON import

Billar accepts historical issued invoices with schema `billar.invoice.import/v1`.

CLI:

```sh
billar invoice import --file examples/imports/INV-2026-00001.json
```

After import, use the CLI-first operational path for review and safe metadata corrections:

```sh
billar invoice inspect --id inv_123 --format json
billar invoice update-metadata --id inv_123 --invoice-date 2026-05-01 --payment-terms "Net 15"
```

`update-metadata` preserves imported identity fields (`import_source`, `imported_at`, external/imported numbers) and invoice financials. Omitted or empty optional flags mean unchanged.

MCP tool: `invoice_import` with a required `payload` object using the same JSON shape.

Operational note: MCP remains available for import/read flows, but routine billing writes should use the trusted CLI/app service path. Direct SQLite edits are emergency repair-only and require explicit operator approval; MCP billing writes are not trusted in this posture.

V1 requires existing customer/issuer profile IDs (or customer resolution by existing tax ID/legal name). It does not parse PDFs or auto-create profiles. Imported line `amount_minor` is authoritative; `quantity_display` and `unit_price_display` are preserved for display/PDF rendering.
