CREATE TABLE IF NOT EXISTS ledger_entries (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    idempotency_key TEXT,
    transaction_id TEXT,
    amount INT,
    type TEXT,
    created_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS balance (
    user_id TEXT PRIMARY KEY,
    current_balance BIGINT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);