app-name=athleton

include .env
export

# Database settings from .env file, with defaults
DATABASE_DRIVER   ?= postgres
DATABASE_HOST     ?= localhost
DATABASE_PORT     ?= 5432
DATABASE_USERNAME ?= postgres
DATABASE_PASSWORD ?= postgres
DATABASE_DATABASE ?= athleton
DATABASE_SSLMODE  ?= disable
name    		  ?= new_migration

# Construct database URL for Atlas
DATABASE_URL := "$(DATABASE_DRIVER)://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_DATABASE)?sslmode=$(DATABASE_SSLMODE)"

dep:
	go mod tidy

vendor:
	go mod vendor

dev:
	go build -o bin/${app-name} cmd/main.go
	./bin/${app-name}

run:
	@echo "Running the application..."
	go run cmd/main.go

# Usage: make migrate-create name=my_migration_name
migrate-create:
	@echo "Creating new migration file..."
	atlas migrate diff $(name) --env gorm 

migrate-up:
	@echo "Applying migrations..."
	@atlas migrate apply --dir file://database/migrations?format=golang-migrate --url "$(DATABASE_URL)"

migrate-down:
	@echo "Reverting migrations..."
	@atlas migrate down --env gorm --dir file://database/migrations?format=golang-migrate --url "$(DATABASE_URL)"

migrate-status:
	@echo "Checking migration status..."
	@atlas migrate status --dir file://database/migrations?format=golang-migrate --url "$(DATABASE_URL)"

migrate-hash:
	@echo "Re-hashing migration files..."
	@atlas migrate hash --dir file://database/migrations?format=golang-migrate

debug:
	@echo "MIGRATION_NAME: $(name)"

swag:
	swag fmt
	swag init -g cmd/main.go

swag-format:
	swag fmt

test:
	go test ./... -coverprofile cp.out

test-html:
	go test $(go list ./... | grep -v /mock/) -coverprofile cp.out
	go tool cover -html=cp.out

module:
	go run ./cmd/generate/main.go -name $(name) -model -dto

# Regenerate GORM CLI field helpers from internal/models into internal/generated.
# Requires: go install gorm.io/cli/gorm@latest
gorm-gen:
	@echo "Generating GORM CLI field helpers..."
	go generate ./internal/models/...

seed:
	go run ./database/seeder/main.go

build:
	set GOOS=linux&& set GOARCH=amd64&& go build -o bin/${app-name} cmd/main.go
