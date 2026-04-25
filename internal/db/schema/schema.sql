CREATE TABLE expenses (
    id               SERIAL PRIMARY KEY,
    description      TEXT NOT NULL,
    amount           DECIMAL(10, 2) NOT NULL,
    currency         TEXT NOT NULL,
    charged_amount   DECIMAL(10, 2),
    charged_currency TEXT,
    payment_method   TEXT NOT NULL DEFAULT 'cash',
    category         TEXT,
    transfer_id      INTEGER REFERENCES transfers (id) ON DELETE CASCADE,
    date             TIMESTAMPTZ(3) NOT NULL,
    created_at       TIMESTAMPTZ(3) NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ(3) NOT NULL DEFAULT NOW()
);

CREATE TABLE transfers (
    id                  SERIAL PRIMARY KEY,
    from_amount         DECIMAL(12, 2) NOT NULL,
    from_currency       TEXT NOT NULL,
    to_amount           DECIMAL(12, 2) NOT NULL,
    to_currency         TEXT NOT NULL,
    commission          DECIMAL(12, 2) NOT NULL DEFAULT 0,
    commission_currency TEXT,
    from_payment_method TEXT,
    to_payment_method   TEXT,
    note                TEXT,
    date                TIMESTAMPTZ(3) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE accounts (
    id                   SERIAL PRIMARY KEY,
    name                 TEXT NOT NULL,
    currency             TEXT NOT NULL,
    payment_method_code  TEXT,
    initial_balance      DECIMAL(12, 2) NOT NULL DEFAULT 0,
    initial_date         TEXT NOT NULL,
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
