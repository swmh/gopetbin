CREATE TABLE "pastes" (
	"id" text PRIMARY KEY,
	"name" text NOT NULL,
	"expire_at" timestamp NOT NULL,
	"remaining_reads" int,
	"created_at" timestamp NULL DEFAULT (now() AT TIME ZONE 'utc'::text)
);
