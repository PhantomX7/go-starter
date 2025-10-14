-- reverse: create index "idx_users_deleted_at" to table: "users"
DROP INDEX "public"."idx_users_deleted_at";
-- reverse: create "users" table
DROP TABLE "public"."users";
-- reverse: create enum type "user_role"
DROP TYPE "public"."user_role";
