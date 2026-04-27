-- Create "incomes" table
CREATE TABLE "incomes" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "date" timestamptz NOT NULL,
  "amount" numeric(18,8) NOT NULL,
  "currency" text NOT NULL,
  "charged_amount" numeric(18,8) NULL,
  "charged_currency" text NULL,
  "account_id" uuid NOT NULL,
  "category_id" uuid NULL,
  "description" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incomes_account_id_fkey" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "incomes_category_id_fkey" FOREIGN KEY ("category_id") REFERENCES "categories" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "incomes_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_incomes_account_id" to table: "incomes"
CREATE INDEX "idx_incomes_account_id" ON "incomes" ("account_id");
-- Create index "idx_incomes_date" to table: "incomes"
CREATE INDEX "idx_incomes_date" ON "incomes" ("date");
