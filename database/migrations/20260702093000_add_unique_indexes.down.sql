-- reverse: replace full unique index "idx_admin_roles_name" with a partial one
DROP INDEX "idx_admin_roles_name";
CREATE UNIQUE INDEX "idx_admin_roles_name" ON "admin_roles" ("name");
-- reverse: create index "idx_refresh_tokens_user_id" to table: "refresh_tokens"
DROP INDEX "idx_refresh_tokens_user_id";
-- reverse: create index "idx_refresh_tokens_token" to table: "refresh_tokens"
DROP INDEX "idx_refresh_tokens_token";
-- reverse: create index "idx_configs_key" to table: "configs"
DROP INDEX "idx_configs_key";
-- reverse: create index "idx_users_email" to table: "users"
DROP INDEX "idx_users_email";
-- reverse: create index "idx_users_username" to table: "users"
DROP INDEX "idx_users_username";
