CREATE TABLE budget_categories (
    id          UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100)    NOT NULL,
    group_name  VARCHAR(20)     NOT NULL CHECK (group_name IN ('Fixed','Flexible','Non-Monthly')),
    budgeted    DECIMAL(12,2)   NOT NULL CHECK (budgeted >= 0),
    color       VARCHAR(50)     NOT NULL DEFAULT 'var(--chart-1)',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_budget_user ON budget_categories(user_id);
