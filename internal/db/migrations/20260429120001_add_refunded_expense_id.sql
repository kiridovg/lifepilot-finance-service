-- Modify "incomes" table
ALTER TABLE "incomes" ADD COLUMN "refunded_expense_id" uuid NULL, ADD CONSTRAINT "incomes_refunded_expense_id_fkey" FOREIGN KEY ("refunded_expense_id") REFERENCES "expenses" ("id") ON UPDATE NO ACTION ON DELETE SET NULL;
