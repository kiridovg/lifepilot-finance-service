#!/usr/bin/env python3
"""Generate SQL for Revolut UKR (Жена) account migration."""

REVOLUT_UKR = 'b59878ea-3932-4397-a50b-4133e0e52209'
REVOLUT_EUR = 'b04ce008-47ab-4c66-9ded-1a99e5be2112'  # new account (from Mar 6)
BCR_EUR     = 'ecd9ea62-8754-4fd5-874d-cdfe5fd792f8'
WISE_EUR    = 'fa35c842-a6e4-4728-bf54-49549476646c'
ZHENA       = '00000000-0000-0000-0000-000000000002'

CAT_INCOME   = '00000000-0000-0000-0000-000000000005'
CAT_RENT     = 'b8bd4d37-036a-44c7-864c-6f5c5f4683bf'
CAT_TRADING  = '07d540aa-0785-45fa-9ab1-3759a4ebc84e'
CAT_BANKFEES = '00000000-0000-0000-0000-000000000001'

def q(s): return f"'{s}'"

# All credits → incomes
INCOMES = [
    ("2025-03-06", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-03-08", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-03-11", 55.09,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-03-13", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-03-14", 30.00,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2025-03-15", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-03-19", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-03-20", 30.00,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2025-03-25", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-03-26", 30.00,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2025-03-26", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-03-27", 27.55,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-03-31", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-04-02",  1.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-04-02", 32.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-04-03", 30.00,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2025-04-03", 55.55,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-04-05", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-04-07", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-04-09", 32.70,  "Платеж от YANINA PONOMARENKO"),
    ("2025-04-09", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-04-09", 30.00,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2025-04-13", 35.00,  "Платеж от R.A. REIJNS EO Y. REIJNS"),
    ("2025-04-17", 55.10,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-04-17", 30.00,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2025-04-22", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-04-24", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-04-24", 30.70,  "Платеж от YANINA PONOMARENKO"),
    ("2025-04-28", 35.00,  "Платеж от R.A. REIJNS EO Y. REIJNS"),
    ("2025-04-29", 31.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-04-29",140.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-05-01", 55.10,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-05-02", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-05-05", 35.00,  "Платеж от R.A. REIJNS EO Y. REIJNS"),
    ("2025-05-09", 31.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-05-10", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-05-13", 35.00,  "Платеж от R.A. REIJNS EO Y. REIJNS"),
    ("2025-05-14", 55.10,  "Платеж от WISE"),
    ("2025-05-15", 31.20,  "Платеж от YANINA PONOMARENKO"),
    ("2025-05-19",100.00,  "Пополнение Apple Pay *2932"),
    ("2025-05-19",100.00,  "Пополнение Mono*kyreieva Elina"),
    ("2025-05-19",500.00,  "Пополнение Mono*kyreieva Elina"),
    ("2025-05-20", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-05-23",137.75,  "Платеж от WISE"),
    ("2025-05-27", 35.00,  "Платеж от R.A. REIJNS EO Y. REIJNS"),
    ("2025-05-30", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-06-06", 35.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-06-10", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-06-13", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-06-16", 30.00,  "Перевод, отправитель: TETIANA GLADKOVA"),
    ("2025-06-17", 32.00,  "Платеж от LIUDMYLA RASEVYCH"),
    ("2025-06-18", 30.90,  "Платеж от YANINA PONOMARENKO"),
    ("2025-06-19", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-06-24", 30.50,  "Платеж от YANINA PONOMARENKO"),
    ("2025-06-25",1000.00, "Пополнение Mono*kyreieva Elina"),
    ("2025-06-26", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-06-29",125.00,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-07-03", 30.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-07-03", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-07-09", 30.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-07-10", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-07-17", 30.00,  "Платеж от YANINA PONOMARENKO"),
    ("2025-07-18", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-07-26", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-08-01", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-08-04",1000.00, "Пополнение Mono*kyreieva Elina"),
    ("2025-08-08", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-08-16", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-08-23", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-08-25", 54.00,  "Платеж от WISE / KHRYSTYNA MORYS"),
    ("2025-09-13", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-09-17", 54.00,  "Платеж от WISE"),
    ("2025-09-20", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-09-28", 54.00,  "Платеж от WISE"),
    ("2025-09-29", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-10-01",142.00,  "Платеж от YULIIA ROMANENKO"),
    ("2025-10-03", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-10-13", 54.00,  "Платеж от WISE"),
    ("2025-10-18", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-10-22", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-10-29",145.00,  "Платеж от YULIIA ROMANENKO"),
    ("2025-10-30", 54.00,  "Платеж от WISE"),
    ("2025-10-31", 31.00,  "Платеж от MEVR. OLENA HONTARENKO"),
    ("2025-11-01", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-11-08", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-11-15", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-11-17", 31.00,  "Платеж от MEVR. OLENA HONTARENKO"),
    ("2025-11-22", 32.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-11-22",  3.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-11-24", 31.00,  "Платеж от MEVR. OLENA HONTARENKO"),
    ("2025-12-03", 34.00,  "Платеж от NATALIIA MACHYANOVA"),
    ("2025-12-10", 67.00,  "Платеж от NATALIIA MACHYANOVA"),
    ("2025-12-13", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-12-17", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2025-12-22", 31.00,  "Платеж от MEVR. OLENA HONTARENKO"),
    ("2025-12-23", 35.00,  "Платеж от YULIIA ROMANENKO"),
    ("2025-12-24",180.00,  "Платеж от YULIIA ROMANENKO"),
    ("2026-01-04", 54.00,  "Платеж от KHRYSTYNA MORYS"),
    ("2026-01-09", 96.00,  "Платеж от NATALIIA MACHYANOVA"),
    ("2026-01-10", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2026-01-17", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2026-01-20", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2026-01-23", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2026-01-26", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2026-01-29",180.00,  "Платеж от YULIIA ROMANENKO"),
    ("2026-01-30", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2026-02-01", 33.00,  "Платеж от ZHYVCHUK YELYZAVETA"),
    ("2026-02-04", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2026-02-05", 63.00,  "Платеж от NATALIIA MACHYANOVA"),
    ("2026-02-06", 52.00,  "Платеж от KHRYSTYNA MORYS"),
    ("2026-02-07", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2026-02-11", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2026-02-13", 50.00,  "Возврат OnlyChain Fintech Limited"),
    ("2026-02-14", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2026-02-18", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
    ("2026-02-21", 35.00,  "Перевод, отправитель: Anton Schneider"),
    ("2026-02-24", 31.80,  "Перевод, отправитель: ANDRII GLADKOV"),
]

# Expenses: (date, amount, currency, category_id, description, charged_amount, charged_currency)
EXPENSES = [
    ("2025-06-04", 1000.00, "EUR", None,        "POMPILIU-IONUT MIRON",        None, None),
    ("2025-06-04",  200.00, "EUR", None,        "PAUL-MIHAI POPA",             None, None),
    ("2025-07-04",  397.13, "EUR", CAT_RENT,   "Аренда · Cristina Miron",     2000.00, "RON"),
    ("2025-07-10",   52.28, "EUR", CAT_RENT,   "Аренда доп. · Cristina Miron", 264.00, "RON"),
    ("2025-08-04",  396.02, "EUR", CAT_RENT,   "Аренда · Cristina Miron",     2000.00, "RON"),
    ("2025-08-14",   86.22, "EUR", CAT_RENT,   "Аренда доп. · Cristina Miron", 434.31, "RON"),
    ("2025-09-04",  395.89, "EUR", CAT_RENT,   "Аренда · Cristina Miron",     2000.00, "RON"),
    ("2025-09-13",   82.59, "EUR", CAT_RENT,   "Аренда доп. · Cristina Miron", 412.27, "RON"),
    ("2025-10-04",  398.93, "EUR", CAT_RENT,   "Аренда · Cristina Miron",     2000.00, "RON"),
    ("2025-10-11",   52.63, "EUR", CAT_RENT,   "Аренда доп. · Cristina Miron",  264.00, "RON"),
    ("2026-02-07",   50.00, "EUR", CAT_BANKFEES, "OnlyChain Fintech Limited",  None, None),
    ("2026-02-23", 1000.00, "EUR", CAT_TRADING, "Interactive Brokers (инвестиции)", None, None),
    ("2026-02-26",  500.00, "EUR", CAT_TRADING, "Interactive Brokers (инвестиции)", None, None),
]

def main():
    print("BEGIN;")
    print()
    print("-- Set initial balance for Revolut UKR")
    print(f"UPDATE accounts SET initial_balance = 704.44, initial_date = '2025-03-06'")
    print(f"WHERE id = '{REVOLUT_UKR}';")
    print()

    print("-- Move Feb-2026 transfers to Wise from new Revolut EUR → Revolut UKR")
    print(f"UPDATE transfers SET from_account_id = '{REVOLUT_UKR}'")
    print(f"WHERE from_account_id = '{REVOLUT_EUR}' AND date < '2026-03-06';")
    print()

    print(f"-- Insert {len(INCOMES)} incomes")
    for date, amount, desc in INCOMES:
        safe_desc = desc.replace("'", "''")
        print(f"INSERT INTO incomes (user_id, account_id, date, amount, currency, category_id, description)")
        print(f"VALUES ('{ZHENA}', '{REVOLUT_UKR}', '{date}', {amount}, 'EUR', '{CAT_INCOME}', '{safe_desc}');")
    print()

    print("COMMIT;")

if __name__ == '__main__':
    main()
