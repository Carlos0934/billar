ALTER TABLE invoices ADD COLUMN invoice_date INTEGER;
ALTER TABLE invoices ADD COLUMN payment_terms TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN payment_communication TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN import_source TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN external_number TEXT NOT NULL DEFAULT '';
ALTER TABLE invoices ADD COLUMN imported_at INTEGER;

DROP INDEX IF EXISTS idx_invoice_lines_invoice_id;

ALTER TABLE invoice_lines RENAME TO invoice_lines_old;

CREATE TABLE invoice_lines (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    service_agreement_id TEXT,
    time_entry_id TEXT,
    description TEXT NOT NULL DEFAULT '',
    quantity_min INTEGER NOT NULL DEFAULT 0,
    unit_rate_amount INTEGER NOT NULL,
    unit_rate_currency TEXT NOT NULL,
    amount_minor INTEGER,
    tax_minor INTEGER NOT NULL DEFAULT 0,
    unit_price_display TEXT NOT NULL DEFAULT '',
    quantity_display TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE,
    FOREIGN KEY (service_agreement_id) REFERENCES service_agreements(id) ON DELETE CASCADE,
    FOREIGN KEY (time_entry_id) REFERENCES time_entries(id) ON DELETE SET NULL
);

INSERT INTO invoice_lines (id, invoice_id, service_agreement_id, time_entry_id, description, quantity_min, unit_rate_amount, unit_rate_currency)
SELECT id, invoice_id, service_agreement_id, time_entry_id, description, quantity_min, unit_rate_amount, unit_rate_currency
FROM invoice_lines_old;

DROP TABLE invoice_lines_old;

CREATE INDEX idx_invoice_lines_invoice_id ON invoice_lines(invoice_id);
CREATE UNIQUE INDEX uq_invoices_invoice_number ON invoices(invoice_number) WHERE invoice_number != '';
