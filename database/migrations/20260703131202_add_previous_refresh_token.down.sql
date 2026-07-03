-- reverse: create index "idx_refresh_tokens_previous_token_hash" to table: "refresh_tokens"
DROP INDEX "idx_refresh_tokens_previous_token_hash";
-- reverse: modify "refresh_tokens" table
ALTER TABLE "refresh_tokens" DROP COLUMN "previous_token_hash";
