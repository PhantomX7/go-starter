-- create enum type "user_role"
CREATE TYPE "public"."user_role" AS ENUM ('user', 'admin', 'reseller');
-- create "users" table
CREATE TABLE "public"."users" (
  "id" bigserial NOT NULL,
  "username" character varying(255) NOT NULL,
  "email" character varying(255) NOT NULL,
  "phone" character varying(255) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "role" "public"."user_role" NOT NULL,
  "password" character varying(255) NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "public"."users" ("deleted_at");
