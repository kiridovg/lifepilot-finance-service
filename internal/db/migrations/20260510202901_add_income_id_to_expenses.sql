-- Modify "expenses" table
ALTER TABLE "expenses" ADD COLUMN "income_id" uuid NULL, ADD CONSTRAINT "fk_expenses_income_id" FOREIGN KEY ("income_id") REFERENCES "incomes" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Create index "idx_expenses_income_id" to table: "expenses"
CREATE INDEX "idx_expenses_income_id" ON "expenses" ("income_id");
