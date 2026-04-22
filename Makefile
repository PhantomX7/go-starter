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
module_name       ?=
model             ?= 1
dto               ?= 1

# Construct database URL for Atlas
DATABASE_URL := "$(DATABASE_DRIVER)://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_DATABASE)?sslmode=$(DATABASE_SSLMODE)"
GENERATOR_FLAGS := $(if $(filter 1 true yes,$(model)),-model,) $(if $(filter 1 true yes,$(dto)),-dto,)

.PHONY: dep vendor dev run migrate-create migrate-up migrate-down migrate-status migrate-hash \
	debug swag swag-format lint lint-fix fmt lint-install hooks-install hooks-uninstall \
	hooks-run test test-html module generate-module gorm-gen seed build

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

lint:
	@echo "Running golangci-lint..."
	@golangci-lint run ./...

lint-fix:
	@echo "Running golangci-lint with --fix..."
	@golangci-lint run --fix ./...

fmt:
	@echo "Running formatters..."
	@golangci-lint fmt

lint-install:
	@echo "Installing golangci-lint v2.11.0..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.0

hooks-install:
	@echo "Installing lefthook and git hooks..."
	@go install github.com/evilmartians/lefthook@latest
	@lefthook install

hooks-uninstall:
	@echo "Uninstalling git hooks..."
	@lefthook uninstall

hooks-run:
	@lefthook run pre-commit --all-files

test:
	go test ./... -coverprofile cp.out

test-html:
	go test $(go list ./... | grep -v /mock/) -coverprofile cp.out
	go tool cover -html=cp.out

# Usage: make generate-module module_name=inventory_item [model=1] [dto=1]
generate-module:
	$(if $(strip $(module_name)),,$(error Usage: make generate-module module_name=<feature_name> [model=1] [dto=1]))
	@echo "Generating module '$(module_name)' with flags: $(GENERATOR_FLAGS)"
	go run ./cmd/generate/main.go -name $(module_name) $(GENERATOR_FLAGS)

module: generate-module

# Regenerate GORM CLI field helpers from internal/models into internal/generated.
# Requires: go install gorm.io/cli/gorm@latest
gorm-gen:
	@echo "Generating GORM CLI field helpers..."
	go generate ./internal/models/...

seed:
	go run ./database/seeder/main.go

build:
	set GOOS=linux&& set GOARCH=amd64&& go build -o bin/${app-name} cmd/main.go
