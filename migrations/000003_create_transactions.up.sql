CREATE TABLE transactions (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    merchant   VARCHAR(255)  NOT NULL,
    category   VARCHAR(100)  NOT NULL,
    note       TEXT          NOT NULL DEFAULT '',
    date       DATE          NOT NULL,
    amount     DECIMAL(12,2) NOT NULL CHECK (amount > 0),
    type       VARCHAR(10)   NOT NULL CHECK (type IN ('income','expense')),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ   NULL
);

CREATE INDEX idx_tx_user_date     ON transactions (user_id, date DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_type     ON transactions (user_id, type)      WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_user_category ON transactions (user_id, category)  WHERE deleted_at IS NULL;
