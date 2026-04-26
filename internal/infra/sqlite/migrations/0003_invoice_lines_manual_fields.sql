DROP INDEX IF EXISTS idx_invoice_lines_invoice_id;

ALTER TABLE invoice_lines RENAME TO invoice_lines_old;

CREATE TABLE invoice_lines (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    service_agreement_id TEXT NOT NULL,
    time_entry_id TEXT,
    description TEXT NOT NULL DEFAULT '',
    quantity_min INTEGER NOT NULL DEFAULT 0,
    unit_rate_amount INTEGER NOT NULL,
    unit_rate_currency TEXT NOT NULL,
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE,
    FOREIGN KEY (service_agreement_id) REFERENCES service_agreements(id) ON DELETE CASCADE,
    FOREIGN KEY (time_entry_id) REFERENCES time_entries(id) ON DELETE SET NULL
);

INSERT INTO invoice_lines (id, invoice_id, service_agreement_id, time_entry_id, description, quantity_min, unit_rate_amount, unit_rate_currency)
SELECT il.id,
       il.invoice_id,
       il.service_agreement_id,
       il.time_entry_id,
       COALESCE(te.description, ''),
       COALESCE(te.hours * 60 / 10000, 0),
       il.unit_rate_amount,
       il.unit_rate_currency
FROM invoice_lines_old il
LEFT JOIN time_entries te ON te.id = il.time_entry_id;

DROP TABLE invoice_lines_old;

CREATE INDEX idx_invoice_lines_invoice_id ON invoice_lines(invoice_id);
