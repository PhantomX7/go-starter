app-name=starter

include .env
export

# Database settings from .env file, with defaults
DATABASE_DRIVER   ?= postgres
DATABASE_HOST     ?= localhost
DATABASE_PORT     ?= 5432
DATABASE_USERNAME ?= postgres
DATABASE_PASSWORD ?=
DATABASE_DATABASE ?= athleton
DATABASE_SSLMODE  ?= disable
MIGRATION_NAME    ?= ""

# Construct database URL for Atlas
DATABASE_URL := "$(DATABASE_DRIVER)://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_DATABASE)?sslmode=$(DATABASE_SSLMODE)"

dep:
	go mod tidy
	go mod vendor

dev:
	go build -o bin/${app-name} cmd/main.go
	./bin/${app-name}

# Usage: make migrate-create name=my_migration_name
migrate-create:
	@echo "Creating new migration file..."
	atlas migrate diff $(MIGRATION_NAME) --env gorm 

migrate-up:
	@echo "Applying migrations..."
	@atlas migrate apply --dir file://database/migrations --url "$(DATABASE_URL)"

migrate-down:
	@echo "Reverting migrations..."
	@atlas migrate down --dir file://database/migrations --url "$(DATABASE_URL)"

migrate-status:
	@echo "Checking migration status..."
	@atlas migrate status --dir file://database/migrations --url "$(DATABASE_URL)"

migrate-hash:
	@echo "Re-hashing migration files..."
	@atlas migrate hash --dir file://database/migrations

debug:
	@echo "MIGRATION_NAME: $(MIGRATION_NAME)"

test:
	go test ./... -coverprofile cp.out

test-html:
	go test $(go list ./... | grep -v /mock/) -coverprofile cp.out
	go tool cover -html=cp.out
seed:
	go run ./seeder/main.go

sync-permission:
	cd ./tools/permgen&& go run main.go

build:
	set GOOS=linux&& set GOARCH=amd64&& go build -o bin/${app-name} cmd/main.go

swag:
	swag init -d app