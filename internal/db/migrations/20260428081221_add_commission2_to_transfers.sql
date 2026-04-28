-- Modify "transfers" table
ALTER TABLE "transfers" ADD COLUMN "commission2" numeric(18,8) NULL, ADD COLUMN "commission2_currency" text NULL;
