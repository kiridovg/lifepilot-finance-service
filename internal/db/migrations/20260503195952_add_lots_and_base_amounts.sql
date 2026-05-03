-- Modify "expenses" table
ALTER TABLE "expenses" ADD COLUMN "base_amount" numeric(18,8) NULL, ADD COLUMN "base_currency" text NULL;
-- Modify "incomes" table
ALTER TABLE "incomes" ADD COLUMN "base_amount" numeric(18,8) NULL, ADD COLUMN "base_currency" text NULL;
-- Modify "transfers" table
ALTER TABLE "transfers" ADD COLUMN "rate" numeric(18,10) NULL;
-- Create "account_lots" table
CREATE TABLE "account_lots" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "account_id" uuid NOT NULL,
  "transfer_id" uuid NULL,
  "original_amount" numeric(18,8) NOT NULL,
  "rate_to_base" numeric(18,10) NOT NULL,
  "remaining" numeric(18,8) NOT NULL,
  "base_currency" text NOT NULL,
  "date" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "account_lots_account_id_fkey" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "account_lots_transfer_id_fkey" FOREIGN KEY ("transfer_id") REFERENCES "transfers" ("id") ON UPDATE NO ACTION ON DELETE SET NULL
);
-- Create index "idx_account_lots_account_date" to table: "account_lots"
CREATE INDEX "idx_account_lots_account_date" ON "account_lots" ("account_id", "date");
-- Create index "idx_account_lots_remaining" to table: "account_lots"
CREATE INDEX "idx_account_lots_remaining" ON "account_lots" ("account_id", "date") WHERE (remaining > (0)::numeric);
