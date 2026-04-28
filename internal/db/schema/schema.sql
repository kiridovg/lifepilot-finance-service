CREATE TABLE currencies (
    code       TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    symbol     TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO currencies (code, name, symbol) VALUES
    ('RON', 'Romanian Leu',       'lei'),
    ('UAH', 'Ukrainian Hryvnia',  '₴'),
    ('EUR', 'Euro',               '€'),
    ('USD', 'US Dollar',          '$'),
    ('PLN', 'Polish Zloty',       'zł'),
    ('HUF', 'Hungarian Forint',   'Ft'),
    ('KZT', 'Kazakhstani Tenge',  '₸');


-- Categories: expense | income | bank-fees | transfer
CREATE TABLE categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('expense', 'income', 'bank-fees', 'transfer')),
    parent_id  UUID REFERENCES categories (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed system categories
INSERT INTO categories (id, name, type) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Bank Fees',  'bank-fees'),
    ('00000000-0000-0000-0000-000000000002', 'Exchange',   'transfer'),
    ('00000000-0000-0000-0000-000000000003', 'Food',       'expense'),
    ('00000000-0000-0000-0000-000000000004', 'Transport',  'expense'),
    ('00000000-0000-0000-0000-000000000005', 'Income',     'income');

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Account = provider + currency (one physical card can have multiple accounts)
CREATE TABLE accounts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users (id),
    name            TEXT    NOT NULL,
    payment_method  TEXT,                              -- "wise" | "kaspi" | "cash" | null
    currency        TEXT    NOT NULL,                  -- "EUR" | "KZT" | "USD" | ...
    initial_balance DECIMAL(18, 8) NOT NULL DEFAULT 0,
    initial_date    DATE    NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Transfers: money movement across system boundary or between own accounts
-- from_account_id NULL = money received from external source (income, deposit return)
-- to_account_id   NULL = money sent to external (outgoing transfer, deposit)
-- fromAmount is the TOTAL debited (includes commission) — already in fromAmount
-- linked_transfer_id: links a deposit to its return (or any paired transfers)
CREATE TABLE transfers (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date                 TIMESTAMPTZ NOT NULL,
    from_account_id      UUID        REFERENCES accounts (id),
    from_amount          DECIMAL(18, 8),
    from_currency        TEXT,
    to_account_id        UUID        REFERENCES accounts (id),
    to_amount            DECIMAL(18, 8) NOT NULL,
    to_currency          TEXT        NOT NULL,
    commission            DECIMAL(18, 8),
    commission_currency   TEXT,
    commission2           DECIMAL(18, 8),
    commission2_currency  TEXT,
    description           TEXT,
    linked_transfer_id   UUID        REFERENCES transfers (id) ON DELETE SET NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (from_account_id IS NOT NULL OR to_account_id IS NOT NULL)
);

-- Expenses: money spent
-- chargedAmount/chargedCurrency: when card currency differs from purchase currency
-- transfer_id: if this is a transfer fee — excluded from balance, shown in bank-fees stats
CREATE TABLE expenses (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users (id),
    date              TIMESTAMPTZ NOT NULL,
    amount            DECIMAL(18, 8) NOT NULL,
    currency          TEXT        NOT NULL,
    charged_amount    DECIMAL(18, 8),
    charged_currency  TEXT,
    account_id        UUID        NOT NULL REFERENCES accounts (id),
    category_id       UUID        REFERENCES categories (id),
    description       TEXT,
    transfer_id       UUID        REFERENCES transfers (id) ON DELETE CASCADE,
    is_refund         BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expenses_account_id   ON expenses (account_id);
CREATE INDEX idx_expenses_date         ON expenses (date);
CREATE INDEX idx_expenses_transfer_id  ON expenses (transfer_id);
CREATE INDEX idx_transfers_from_account  ON transfers (from_account_id);
CREATE INDEX idx_transfers_to_account    ON transfers (to_account_id);
CREATE INDEX idx_transfers_date          ON transfers (date);
CREATE INDEX idx_transfers_linked        ON transfers (linked_transfer_id);

-- Incomes: money received
CREATE TABLE incomes (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL REFERENCES users (id),
    date              TIMESTAMPTZ NOT NULL,
    amount            DECIMAL(18, 8) NOT NULL,
    currency          TEXT        NOT NULL,
    charged_amount    DECIMAL(18, 8),
    charged_currency  TEXT,
    account_id        UUID        NOT NULL REFERENCES accounts (id),
    category_id       UUID        REFERENCES categories (id),
    description       TEXT,
    is_refund         BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incomes_account_id ON incomes (account_id);
CREATE INDEX idx_incomes_date       ON incomes (date);
