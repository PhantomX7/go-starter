-- reverse: create index "idx_refresh_tokens_user_id" to table: "refresh_tokens"
DROP INDEX "idx_refresh_tokens_user_id";
-- reverse: create index "idx_refresh_tokens_token" to table: "refresh_tokens"
DROP INDEX "idx_refresh_tokens_token";
-- reverse: create index "idx_refresh_tokens_previous_token_hash" to table: "refresh_tokens"
DROP INDEX "idx_refresh_tokens_previous_token_hash";
-- reverse: create "refresh_tokens" table
DROP TABLE "refresh_tokens";
-- reverse: create index "idx_logs_user_id" to table: "logs"
DROP INDEX "idx_logs_user_id";
-- reverse: create index "idx_logs_entity_type" to table: "logs"
DROP INDEX "idx_logs_entity_type";
-- reverse: create index "idx_logs_entity_id" to table: "logs"
DROP INDEX "idx_logs_entity_id";
-- reverse: create index "idx_logs_deleted_at" to table: "logs"
DROP INDEX "idx_logs_deleted_at";
-- reverse: create "logs" table
DROP TABLE "logs";
-- reverse: create index "idx_users_username" to table: "users"
DROP INDEX "idx_users_username";
-- reverse: create index "idx_users_email" to table: "users"
DROP INDEX "idx_users_email";
-- reverse: create index "idx_users_deleted_at" to table: "users"
DROP INDEX "idx_users_deleted_at";
-- reverse: create index "idx_users_admin_role_id" to table: "users"
DROP INDEX "idx_users_admin_role_id";
-- reverse: create "users" table
DROP TABLE "users";
-- reverse: create index "idx_configs_key" to table: "configs"
DROP INDEX "idx_configs_key";
-- reverse: create index "idx_configs_deleted_at" to table: "configs"
DROP INDEX "idx_configs_deleted_at";
-- reverse: create "configs" table
DROP TABLE "configs";
-- reverse: create index "idx_admin_roles_name" to table: "admin_roles"
DROP INDEX "idx_admin_roles_name";
-- reverse: create index "idx_admin_roles_deleted_at" to table: "admin_roles"
DROP INDEX "idx_admin_roles_deleted_at";
-- reverse: create "admin_roles" table
DROP TABLE "admin_roles";
-- reverse: create enum type "user_role"
DROP TYPE "user_role";
