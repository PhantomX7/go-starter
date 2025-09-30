-- create "posts" table
CREATE TABLE "public"."posts" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "title" character varying(255) NOT NULL,
  "content" text NOT NULL,
  "description" text NULL,
  "image_url" character varying(255) NULL,
  "is_active" boolean NULL DEFAULT true,
  PRIMARY KEY ("id")
);
-- create index "idx_posts_deleted_at" to table: "posts"
CREATE INDEX "idx_posts_deleted_at" ON "public"."posts" ("deleted_at");
