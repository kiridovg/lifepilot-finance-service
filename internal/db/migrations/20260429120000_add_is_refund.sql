-- Add "is_refund" to "expenses" table
ALTER TABLE "expenses" ADD COLUMN "is_refund" boolean NOT NULL DEFAULT false;
-- Add "is_refund" to "incomes" table
ALTER TABLE "incomes" ADD COLUMN "is_refund" boolean NOT NULL DEFAULT false;
