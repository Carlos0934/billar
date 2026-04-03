CREATE TABLE IF NOT EXISTS customers (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    legal_name TEXT NOT NULL,
    trade_name TEXT NOT NULL DEFAULT '',
    tax_id TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    website TEXT NOT NULL DEFAULT '',
    billing_address TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'USD',
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_customers_legal_name ON customers(legal_name);
CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers(created_at);
