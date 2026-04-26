-- Modify "accounts" table
ALTER TABLE "accounts" ALTER COLUMN "user_id" SET NOT NULL;
-- Modify "expenses" table
ALTER TABLE "expenses" ALTER COLUMN "user_id" SET NOT NULL;
