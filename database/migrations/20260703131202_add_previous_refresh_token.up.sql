-- modify "refresh_tokens" table
ALTER TABLE "refresh_tokens" ADD COLUMN "previous_token_hash" text NULL;
-- create index "idx_refresh_tokens_previous_token_hash" to table: "refresh_tokens"
CREATE INDEX "idx_refresh_tokens_previous_token_hash" ON "refresh_tokens" ("previous_token_hash");
