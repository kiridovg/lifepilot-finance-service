#!/usr/bin/env python3
"""Generate INSERT SQL for 2025 expenses from lifepilot-admin pg_dump."""

import re
import sys

DUMP = '/Users/kiridovg/PhpstormProjects/lifepilot-admin/pg_dump/lifepilot-admin_20260426.sql'

KIRILL = '00000000-0000-0000-0000-000000000001'
ZHENA  = '00000000-0000-0000-0000-000000000002'

# (payment_method, currency) -> (account_id, user_id)
ACCOUNT_MAP = {
    ('monobank',           'EUR'): ('785b14c2-80fd-43be-a2b0-9d56efa85ec4', KIRILL),
    ('monobank',           'USD'): ('4ab8f84e-e9e2-45e9-9821-432c91a3befe', KIRILL),
    ('monobank',           'UAH'): ('32d33c44-8883-4775-a119-2306f4eb5818', ZHENA),
    ('monobank',           'RON'): ('32d33c44-8883-4775-a119-2306f4eb5818', ZHENA),
    ('priv24',             'UAH'): ('313c5d85-9d9b-4bdf-b8d5-0ffb340e98fb', ZHENA),
    ('priv24',             'USD'): ('9971a735-c919-4764-9373-1cabc38a68ee', KIRILL),
    ('priv24',             'RON'): ('313c5d85-9d9b-4bdf-b8d5-0ffb340e98fb', ZHENA),
    ('priv24',             'EUR'): ('313c5d85-9d9b-4bdf-b8d5-0ffb340e98fb', ZHENA),
    ('priv24',             'PLN'): ('313c5d85-9d9b-4bdf-b8d5-0ffb340e98fb', ZHENA),
    ('wise',               'EUR'): ('fa35c842-a6e4-4728-bf54-49549476646c', KIRILL),
    ('wise',               'USD'): ('9d02057f-dd36-49dd-8d00-7eee793e4f2f', KIRILL),
    ('wise',               'RON'): ('fa35c842-a6e4-4728-bf54-49549476646c', KIRILL),
    ('wise',               'UAH'): ('fa35c842-a6e4-4728-bf54-49549476646c', KIRILL),
    ('bcr',                'EUR'): ('ecd9ea62-8754-4fd5-874d-cdfe5fd792f8', ZHENA),
    ('bcr',                'USD'): ('55873c0e-1771-490d-b5ae-5aa55680215d', ZHENA),
    ('bcr',                'RON'): ('ecd9ea62-8754-4fd5-874d-cdfe5fd792f8', ZHENA),
    ('bcr',                'UAH'): ('ecd9ea62-8754-4fd5-874d-cdfe5fd792f8', ZHENA),
    ('banca-transilvania', 'RON'): ('4f9f9565-9907-42b2-a0bc-6a396ef35c6b', KIRILL),
    ('banca-transilvania', 'EUR'): ('4f9f9565-9907-42b2-a0bc-6a396ef35c6b', KIRILL),
    ('banca-transilvania', 'USD'): ('6f14807b-5bbd-40f7-a391-145f6420f031', KIRILL),
    ('abank',              'UAH'): ('a65867a6-788f-4243-b002-6178f21642c2', KIRILL),
    ('abank',              'EUR'): ('73f67951-ca46-438c-a91b-32ddcb2e7d24', KIRILL),
    ('abank',              'RON'): ('73f67951-ca46-438c-a91b-32ddcb2e7d24', KIRILL),
    ('abank',              'USD'): ('73f67951-ca46-438c-a91b-32ddcb2e7d24', KIRILL),
    ('abank',              'PLN'): ('73f67951-ca46-438c-a91b-32ddcb2e7d24', KIRILL),
    ('velobank',           'PLN'): ('05775af6-5605-4e54-8991-ec1337a815ad', KIRILL),
    ('velobank',           'EUR'): ('99ab337c-2789-4d9c-9e20-46cb3901d171', KIRILL),
    ('paypal',             'USD'): ('25f588d7-04c2-4220-b836-6212d83800c7', KIRILL),
    ('paypal',             'EUR'): ('25f588d7-04c2-4220-b836-6212d83800c7', KIRILL),
    ('pumb',               'UAH'): ('03ed9eae-db98-48e5-9475-183d7569d1b2', KIRILL),
    ('pumb',               'EUR'): ('c14abc1e-176d-4cf4-8e15-826ea31ecea5', ZHENA),
    ('revolut',            'EUR'): ('b04ce008-47ab-4c66-9ded-1a99e5be2112', ZHENA),
    ('revolut',            'USD'): ('f3f1f8d9-4780-4066-9ea2-fdde2d3df74d', ZHENA),
    ('revolut',            'RON'): ('b04ce008-47ab-4c66-9ded-1a99e5be2112', ZHENA),
    ('wise',               'HUF'): ('fa35c842-a6e4-4728-bf54-49549476646c', KIRILL),
    ('ukrsibbank',         'UAH'): ('2772944b-0ed9-4794-a098-a5039e6ce7d1', KIRILL),
    ('raiffeisen-bank',    'UAH'): ('95815117-df8d-4bd7-975a-422854b7d98f', KIRILL),
    ('cash',               'EUR'): ('9fa18162-9386-4818-a766-6c63b44fcc76', KIRILL),
    ('cash',               'RON'): ('be31efa9-5e96-41cf-b80c-c203c7b84d5d', KIRILL),
    ('cash',               'USD'): ('8dd48cd0-62a8-4f66-b365-c7613a7ca728', KIRILL),
    ('cash',               'UAH'): ('fde6acd2-f617-441c-bd73-805256effdf5', KIRILL),
    ('cash',               'PLN'): ('05775af6-5605-4e54-8991-ec1337a815ad', KIRILL),
    ('cash',               'KZT'): ('a71a07ce-951e-467e-99b3-809d43837a1e', KIRILL),
}

CATEGORY_MAP = {
    'food':                    '00000000-0000-0000-0000-000000000003',
    'transport-taxi':          '00000000-0000-0000-0000-000000000004',
    'transport-bus':           '00000000-0000-0000-0000-000000000004',
    'transport-plane':         '00000000-0000-0000-0000-000000000004',
    'bank-fees':               '00000000-0000-0000-0000-000000000001',
    'health':                  '0f720ef8-b54e-491e-8c19-9c43a6e59aef',
    'rent':                    'b8bd4d37-036a-44c7-864c-6f5c5f4683bf',
    'utilities':               '168aef5b-74d2-4d1a-a7ea-8295b9b63b8b',
    'taxes-business':          '06c11b45-1e0b-49a5-b890-960343c200d3',
    'hotel':                   '070acf0e-0393-4338-9c53-c370ff5ce1ba',
    '\\N':                     None,
    'other':                   None,
    '\u0441lothing':           '82778be0-2f3d-4ef8-a7cb-1b9e8e6a9b41',  # cyrillic с
    'pharmacy':                'a26eea76-5bd4-48d4-88b4-2c3d6a1951b7',
    'restaurant-cafe':         '18c67e70-3c2d-467e-8c41-f018c36bfc29',
    'postal-services':         None,
    'mobile-services':         '1222d766-c274-4d29-bbc9-d11f07ef5f65',
    'aliexpress':              '0befd122-74ae-4c3a-814b-60fb467ebb87',   # Home
    'electronics-accessories': '0befd122-74ae-4c3a-814b-60fb467ebb87',   # Home
    'trading-services':        '07d540aa-0785-45fa-9ab1-3759a4ebc84e',
    'fuel-passat-b6':          'b754b72f-6f90-4fde-b388-c3693a1746df',
    'car-passat-b6':           '1dfac94f-9546-4e46-a847-57d7c45231ff',
    'subscriptions':           'a5d79578-44ce-40da-83f0-16bbd01c4fbe',
    'tickets':                 '804df8f2-e2e7-4a11-a1a9-157b79ca6162',
    'books':                   'a5d79578-44ce-40da-83f0-16bbd01c4fbe',   # Subscriptions
    'beauty-personal-care':    '0f0d4761-b888-47d5-993c-40c3d6fffa26',
    'home-household':          '0befd122-74ae-4c3a-814b-60fb467ebb87',
    'education-karazin':       '16023f2d-671c-4eb7-baac-52dbb4207882',   # Taxes (closest)
    'hosting':                 '3a0c3bb3-1140-4e73-baff-d689550a45f5',
    'donations':               None,
}

def esc(s):
    if s is None or s == '\\N':
        return 'NULL'
    return "'" + s.replace("'", "''") + "'"

def main():
    with open(DUMP, 'r') as f:
        content = f.read()

    match = re.search(r'COPY public\.expenses.*?FROM stdin;\n(.*?)\\\.', content, re.DOTALL)
    if not match:
        print("ERROR: expenses block not found", file=sys.stderr)
        sys.exit(1)

    rows = [r for r in match.group(1).strip().split('\n') if r]
    rows_2025 = [r for r in rows if r.split('\t')[5][:4] == '2024']

    print("-- 2024 expenses migration")
    print("-- Generated from lifepilot-admin pg_dump")
    print(f"-- Total: {len(rows_2025)} expenses")
    print("BEGIN;")
    print()

    skipped = []
    inserted = 0

    for row in rows_2025:
        p = row.split('\t')
        # id, description, amount, currency, category, date, created_at, updated_at, charged_amount, charged_currency, payment_method
        desc       = p[1]
        amount     = p[2]
        currency   = p[3]
        category   = p[4]
        date       = p[5][:10]  # strip time
        ch_amount  = None if p[8] == '\\N' else p[8]
        ch_curr    = None if p[9] == '\\N' else p[9]
        method     = p[10].strip()

        key = (method, currency)
        if key not in ACCOUNT_MAP:
            skipped.append(f"-- SKIP: {method}/{currency} — {desc[:50]}")
            continue

        account_id, user_id = ACCOUNT_MAP[key]
        cat_id = CATEGORY_MAP.get(category)

        cat_sql = esc(cat_id) if cat_id else 'NULL'
        ch_amt_sql = esc(ch_amount) if ch_amount else 'NULL'
        ch_cur_sql = esc(ch_curr) if ch_curr else 'NULL'

        print(f"INSERT INTO expenses (user_id, account_id, date, amount, currency, category_id, charged_amount, charged_currency, description)")
        print(f"VALUES ({esc(user_id)}, {esc(account_id)}, {esc(date)}, {amount}, {esc(currency)}, {cat_sql}, {ch_amt_sql}, {ch_cur_sql}, {esc(desc)});")
        inserted += 1

    print()
    print("COMMIT;")
    print()
    if skipped:
        print(f"-- SKIPPED {len(skipped)} rows (no account mapping):")
        for s in skipped:
            print(s)
    print(f"-- Inserted: {inserted}, Skipped: {len(skipped)}")

if __name__ == '__main__':
    main()
