-- create enum type "user_role"
CREATE TYPE "user_role" AS ENUM ('user', 'admin', 'root');
-- create "admin_roles" table
CREATE TABLE "admin_roles" (
  "id" bigserial NOT NULL,
  "name" character varying(100) NOT NULL,
  "description" character varying(255) NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- create index "idx_admin_roles_deleted_at" to table: "admin_roles"
CREATE INDEX "idx_admin_roles_deleted_at" ON "admin_roles" ("deleted_at");
-- create index "idx_admin_roles_name" to table: "admin_roles"
CREATE UNIQUE INDEX "idx_admin_roles_name" ON "admin_roles" ("name") WHERE (deleted_at IS NULL);
-- create "configs" table
CREATE TABLE "configs" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "key" character varying(255) NOT NULL,
  "value" text NOT NULL,
  "is_public" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("id")
);
-- create index "idx_configs_deleted_at" to table: "configs"
CREATE INDEX "idx_configs_deleted_at" ON "configs" ("deleted_at");
-- create index "idx_configs_key" to table: "configs"
CREATE UNIQUE INDEX "idx_configs_key" ON "configs" ("key") WHERE (deleted_at IS NULL);
-- create "users" table
CREATE TABLE "users" (
  "id" bigserial NOT NULL,
  "username" character varying(255) NOT NULL,
  "name" character varying(255) NULL,
  "business_name" character varying(255) NULL,
  "email" character varying(255) NOT NULL,
  "phone" character varying(255) NOT NULL,
  "is_active" boolean NOT NULL DEFAULT true,
  "role" "user_role" NOT NULL,
  "admin_role_id" bigint NULL,
  "password" character varying(255) NOT NULL,
  "password_changed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_users_admin_role" FOREIGN KEY ("admin_role_id") REFERENCES "admin_roles" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "idx_users_admin_role_id" to table: "users"
CREATE INDEX "idx_users_admin_role_id" ON "users" ("admin_role_id");
-- create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");
-- create index "idx_users_email" to table: "users"
CREATE UNIQUE INDEX "idx_users_email" ON "users" ("email") WHERE (deleted_at IS NULL);
-- create index "idx_users_username" to table: "users"
CREATE UNIQUE INDEX "idx_users_username" ON "users" ("username") WHERE (deleted_at IS NULL);
-- create "logs" table
CREATE TABLE "logs" (
  "id" bigserial NOT NULL,
  "user_id" bigint NULL,
  "action" character varying(50) NOT NULL,
  "entity_type" character varying(50) NULL,
  "entity_id" bigint NULL,
  "message" text NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "deleted_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_logs_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "idx_logs_deleted_at" to table: "logs"
CREATE INDEX "idx_logs_deleted_at" ON "logs" ("deleted_at");
-- create index "idx_logs_entity_id" to table: "logs"
CREATE INDEX "idx_logs_entity_id" ON "logs" ("entity_id");
-- create index "idx_logs_entity_type" to table: "logs"
CREATE INDEX "idx_logs_entity_type" ON "logs" ("entity_type");
-- create index "idx_logs_user_id" to table: "logs"
CREATE INDEX "idx_logs_user_id" ON "logs" ("user_id");
-- create "refresh_tokens" table
CREATE TABLE "refresh_tokens" (
  "id" text NOT NULL,
  "user_id" bigint NOT NULL,
  "token" text NOT NULL,
  "previous_token_hash" text NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "revoked_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_refresh_tokens_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- create index "idx_refresh_tokens_previous_token_hash" to table: "refresh_tokens"
CREATE INDEX "idx_refresh_tokens_previous_token_hash" ON "refresh_tokens" ("previous_token_hash");
-- create index "idx_refresh_tokens_token" to table: "refresh_tokens"
CREATE UNIQUE INDEX "idx_refresh_tokens_token" ON "refresh_tokens" ("token");
-- create index "idx_refresh_tokens_user_id" to table: "refresh_tokens"
CREATE INDEX "idx_refresh_tokens_user_id" ON "refresh_tokens" ("user_id");
