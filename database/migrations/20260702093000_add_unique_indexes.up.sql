-- create index "idx_users_username" to table: "users" (partial: soft-deleted rows do not block reuse)
CREATE UNIQUE INDEX "idx_users_username" ON "users" ("username") WHERE deleted_at IS NULL;
-- create index "idx_users_email" to table: "users" (partial: soft-deleted rows do not block reuse)
CREATE UNIQUE INDEX "idx_users_email" ON "users" ("email") WHERE deleted_at IS NULL;
-- create index "idx_configs_key" to table: "configs" (partial: soft-deleted rows do not block reuse)
CREATE UNIQUE INDEX "idx_configs_key" ON "configs" ("key") WHERE deleted_at IS NULL;
-- create index "idx_refresh_tokens_token" to table: "refresh_tokens" (table never soft-deletes)
CREATE UNIQUE INDEX "idx_refresh_tokens_token" ON "refresh_tokens" ("token");
-- create index "idx_refresh_tokens_user_id" to table: "refresh_tokens"
CREATE INDEX "idx_refresh_tokens_user_id" ON "refresh_tokens" ("user_id");
-- replace full unique index "idx_admin_roles_name" with a partial one so soft-deleted names become reusable
DROP INDEX "idx_admin_roles_name";
CREATE UNIQUE INDEX "idx_admin_roles_name" ON "admin_roles" ("name") WHERE deleted_at IS NULL;
