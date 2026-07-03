-- add "is_public" to "configs": gates the unauthenticated /public/config
-- surface; default false so visibility is opt-in per row.
ALTER TABLE "configs" ADD COLUMN "is_public" boolean NOT NULL DEFAULT false;
