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

-- Service agreements define billing terms for a customer profile.
-- Multiple agreements can exist per customer profile (e.g. different projects or rates over time).
CREATE TABLE IF NOT EXISTS service_agreements (
    id TEXT PRIMARY KEY,
    customer_profile_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    billing_mode TEXT NOT NULL,
    hourly_rate INTEGER NOT NULL,
    currency TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    valid_from INTEGER,
    valid_until INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (customer_profile_id) REFERENCES customer_profiles(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_service_agreements_customer_profile_id ON service_agreements(customer_profile_id);
CREATE INDEX IF NOT EXISTS idx_service_agreements_created_at ON service_agreements(created_at);

-- Time entries record units of work performed for a customer under a service agreement.
-- customer_profile_id is NOT stored here; it is always derived via JOIN on service_agreements.
CREATE TABLE IF NOT EXISTS time_entries (
    id TEXT PRIMARY KEY,
    service_agreement_id TEXT NOT NULL,
    description TEXT NOT NULL,
    hours INTEGER NOT NULL,
    billable INTEGER NOT NULL DEFAULT 1,
    invoice_id TEXT,
    date INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (service_agreement_id) REFERENCES service_agreements(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_time_entries_service_agreement_id ON time_entries(service_agreement_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_date ON time_entries(date);