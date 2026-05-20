CREATE TABLE recurring_items (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255)    NOT NULL,
    category    VARCHAR(100)    NOT NULL,
    amount      DECIMAL(12,2)   NOT NULL CHECK (amount > 0),
    frequency   VARCHAR(20)     NOT NULL CHECK (frequency IN ('weekly','monthly','annual')),
    kind        VARCHAR(10)     NOT NULL CHECK (kind IN ('expense','income')),
    status      VARCHAR(10)     NOT NULL DEFAULT 'active' CHECK (status IN ('active','paused')),
    color       VARCHAR(20)     NOT NULL DEFAULT '#0ea5e9',
    next_due    DATE            NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recurring_user ON recurring_items(user_id);
CREATE INDEX idx_recurring_due  ON recurring_items(user_id, next_due) WHERE status = 'active';
