CREATE TABLE IF NOT EXISTS quotes (
    id          TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL REFERENCES customer_profiles(id) ON DELETE RESTRICT,
    status      TEXT NOT NULL,
    currency    TEXT NOT NULL,
    notes       TEXT NOT NULL DEFAULT '',
    sent_at     INTEGER,
    accepted_at INTEGER,
    rejected_at INTEGER,
    expired_at  INTEGER,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL,
    CHECK (status IN ('draft', 'sent', 'accepted', 'rejected', 'expired'))
);

CREATE INDEX IF NOT EXISTS idx_quotes_customer_status ON quotes(customer_id, status);

CREATE TABLE IF NOT EXISTS quote_lines (
    id                   TEXT PRIMARY KEY,
    quote_id             TEXT NOT NULL REFERENCES quotes(id) ON DELETE CASCADE,
    service_agreement_id TEXT NOT NULL REFERENCES service_agreements(id) ON DELETE RESTRICT,
    description          TEXT NOT NULL,
    quantity_min         INTEGER NOT NULL,
    unit_rate_amount     INTEGER NOT NULL,
    unit_rate_currency   TEXT NOT NULL,
    created_at           INTEGER NOT NULL,
    CHECK (quantity_min > 0),
    CHECK (unit_rate_amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_quote_lines_quote_id ON quote_lines(quote_id);
