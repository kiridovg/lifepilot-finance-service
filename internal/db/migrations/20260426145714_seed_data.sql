-- Seed currencies
INSERT INTO currencies (code, name, symbol) VALUES
    ('RON', 'Romanian Leu',      'lei'),
    ('UAH', 'Ukrainian Hryvnia', '₴'),
    ('EUR', 'Euro',              '€'),
    ('USD', 'US Dollar',         '$'),
    ('PLN', 'Polish Zloty',      'zł'),
    ('HUF', 'Hungarian Forint',  'Ft'),
    ('KZT', 'Kazakhstani Tenge', '₸')
ON CONFLICT (code) DO NOTHING;

-- Seed system categories
INSERT INTO categories (id, name, type) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Bank Fees', 'bank-fees'),
    ('00000000-0000-0000-0000-000000000002', 'Exchange',  'transfer'),
    ('00000000-0000-0000-0000-000000000003', 'Food',      'expense'),
    ('00000000-0000-0000-0000-000000000004', 'Transport', 'expense'),
    ('00000000-0000-0000-0000-000000000005', 'Income',    'income')
ON CONFLICT (id) DO NOTHING;

-- accounts are seeded after users table is created (see seed_users_accounts migration)