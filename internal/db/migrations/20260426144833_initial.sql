-- Create "accounts" table
CREATE TABLE "accounts" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "payment_method" text NULL,
  "currency" text NOT NULL,
  "initial_balance" numeric(18,8) NOT NULL DEFAULT 0,
  "initial_date" date NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "notes" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "currencies" table
CREATE TABLE "currencies" (
  "code" text NOT NULL,
  "name" text NOT NULL,
  "symbol" text NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("code")
);
-- Create "categories" table
CREATE TABLE "categories" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" text NOT NULL,
  "type" text NOT NULL,
  "parent_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "categories_parent_id_fkey" FOREIGN KEY ("parent_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "categories_type_check" CHECK (type = ANY (ARRAY['expense'::text, 'income'::text, 'bank-fees'::text, 'transfer'::text]))
);
-- Create "transfers" table
CREATE TABLE "transfers" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "date" timestamptz NOT NULL,
  "from_account_id" uuid NULL,
  "from_amount" numeric(18,8) NULL,
  "from_currency" text NULL,
  "to_account_id" uuid NULL,
  "to_amount" numeric(18,8) NOT NULL,
  "to_currency" text NOT NULL,
  "commission" numeric(18,8) NULL,
  "commission_currency" text NULL,
  "description" text NULL,
  "linked_transfer_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "transfers_from_account_id_fkey" FOREIGN KEY ("from_account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "transfers_linked_transfer_id_fkey" FOREIGN KEY ("linked_transfer_id") REFERENCES "transfers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "transfers_to_account_id_fkey" FOREIGN KEY ("to_account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "transfers_check" CHECK ((from_account_id IS NOT NULL) OR (to_account_id IS NOT NULL))
);
-- Create index "idx_transfers_date" to table: "transfers"
CREATE INDEX "idx_transfers_date" ON "transfers" ("date");
-- Create index "idx_transfers_from_account" to table: "transfers"
CREATE INDEX "idx_transfers_from_account" ON "transfers" ("from_account_id");
-- Create index "idx_transfers_linked" to table: "transfers"
CREATE INDEX "idx_transfers_linked" ON "transfers" ("linked_transfer_id");
-- Create index "idx_transfers_to_account" to table: "transfers"
CREATE INDEX "idx_transfers_to_account" ON "transfers" ("to_account_id");
-- Create "expenses" table
CREATE TABLE "expenses" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "date" timestamptz NOT NULL,
  "amount" numeric(18,8) NOT NULL,
  "currency" text NOT NULL,
  "charged_amount" numeric(18,8) NULL,
  "charged_currency" text NULL,
  "account_id" uuid NOT NULL,
  "category_id" uuid NULL,
  "description" text NULL,
  "transfer_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "expenses_account_id_fkey" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "expenses_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "expenses_transfer_id_fkey" FOREIGN KEY ("transfer_id") REFERENCES "transfers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "idx_expenses_account_id" to table: "expenses"
CREATE INDEX "idx_expenses_account_id" ON "expenses" ("account_id");
-- Create index "idx_expenses_date" to table: "expenses"
CREATE INDEX "idx_expenses_date" ON "expenses" ("date");
-- Create index "idx_expenses_transfer_id" to table: "expenses"
CREATE INDEX "idx_expenses_transfer_id" ON "expenses" ("transfer_id");
