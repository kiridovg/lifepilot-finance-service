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

-- Seed accounts
INSERT INTO accounts (name, payment_method, currency, initial_balance, initial_date) VALUES
    ('Monobank',           'monobank',           'UAH', 0, '2026-01-01'),
    ('Monobank',           'monobank',           'USD', 0, '2026-01-01'),
    ('Privat24',           'privat24',           'UAH', 0, '2026-01-01'),
    ('Privat24',           'privat24',           'USD', 0, '2026-01-01'),
    ('Wise',               'wise',               'EUR', 0, '2026-01-01'),
    ('Wise',               'wise',               'USD', 0, '2026-01-01'),
    ('Revolut',            'revolut',            'EUR', 0, '2026-01-01'),
    ('Revolut',            'revolut',            'USD', 0, '2026-01-01'),
    ('BCR',                'bcr',                'EUR', 0, '2026-01-01'),
    ('BCR',                'bcr',                'USD', 0, '2026-01-01'),
    ('ABank',              'abank',              'EUR', 0, '2026-01-01'),
    ('ABank',              'abank',              'UAH', 0, '2026-01-01'),
    ('VeloBank',           'velobank',           'EUR', 0, '2026-01-01'),
    ('VeloBank',           'velobank',           'PLN', 0, '2026-01-01'),
    ('PayPal',             'paypal',             'USD', 0, '2026-01-01'),
    ('PUMB',               'pumb',               'UAH', 0, '2026-01-01'),
    ('Ukrsibbank',         'ukrsibbank',         'UAH', 0, '2026-01-01'),
    ('Banca Transilvania', 'banca-transilvania', 'USD', 0, '2026-01-01'),
    ('Banca Transilvania', 'banca-transilvania', 'RON', 0, '2026-01-01'),
    ('Raiffeisen Bank',    'raiffeisen',         'UAH', 0, '2026-01-01'),
    ('Kaspi',              'kaspi',              'KZT', 0, '2026-01-01'),
    ('Tele2',              'tele2',              'KZT', 0, '2026-01-01');