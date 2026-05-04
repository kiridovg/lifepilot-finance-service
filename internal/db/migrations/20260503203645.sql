-- Modify "account_lots" table
ALTER TABLE "account_lots" DROP CONSTRAINT "account_lots_transfer_id_fkey", ADD CONSTRAINT "account_lots_transfer_id_fkey" FOREIGN KEY ("transfer_id") REFERENCES "transfers" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
