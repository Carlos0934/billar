-- Legal entities hold shared identity and contact information.
-- Both issuers (billing operators) and customers reference a legal entity.
CREATE TABLE IF NOT EXISTS legal_entities (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    legal_name TEXT NOT NULL,
    trade_name TEXT NOT NULL DEFAULT '',
    tax_id TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    website TEXT NOT NULL DEFAULT '',
    billing_address TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_legal_entities_legal_name ON legal_entities(legal_name);
CREATE INDEX IF NOT EXISTS idx_legal_entities_created_at ON legal_entities(created_at);

-- Issuer profiles represent the billing operator (the user's own company).
-- There is typically one issuer profile per installation.
-- The UNIQUE constraint on legal_entity_id enforces 1:1 relationship.
CREATE TABLE IF NOT EXISTS issuer_profiles (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL UNIQUE,
    default_currency TEXT NOT NULL DEFAULT 'USD',
    default_notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (legal_entity_id) REFERENCES legal_entities(id) ON DELETE CASCADE
);

-- Customer profiles represent clients to be billed.
-- References a legal entity for identity/contact data.
-- The UNIQUE constraint on legal_entity_id enforces 1:1 relationship.
CREATE TABLE IF NOT EXISTS customer_profiles (
    id TEXT PRIMARY KEY,
    legal_entity_id TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    default_currency TEXT NOT NULL DEFAULT 'USD',
    notes TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (legal_entity_id) REFERENCES legal_entities(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_customer_profiles_status ON customer_profiles(status);
CREATE INDEX IF NOT EXISTS idx_customer_profiles_created_at ON customer_profiles(created_at);