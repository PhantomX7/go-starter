# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=athleton
BINARY_UNIX=$(BINARY_NAME)_unix

# Build the main application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/main.go

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Tidy dependencies
deps:
	$(GOMOD) tidy
	$(GOMOD) download

# Run the application
run:
	$(GOCMD) run ./cmd/main.go

# Generate a new module (usage: make generate-module name=ModuleName)
generate-module:
	@if [ -z "$(name)" ]; then \
		echo "Error: Please provide a module name. Usage: make generate-module name=ModuleName"; \
		exit 1; \
	fi
	$(GOCMD) run ./cmd/generate/main.go -name $(name)

# Generate a new module with model (usage: make generate-full name=ModuleName)
generate-full:
	@if [ -z "$(name)" ]; then \
		echo "Error: Please provide a module name. Usage: make generate-full name=ModuleName"; \
		exit 1; \
	fi
	$(GOCMD) run ./cmd/generate/main.go -name $(name) -model

# Show generator help
generate-help:
	$(GOCMD) run ./cmd/generate/main.go -help

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/main.go

# Docker build
docker-build:
	docker build -t $(BINARY_NAME) .

# Format code
fmt:
	$(GOCMD) fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Install dependencies
install-deps:
	$(GOGET) -u ./...

# Database migrations (if using migrate tool)
migrate-up:
	migrate -path database/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path database/migrations -database "$(DATABASE_URL)" down

# Generate swagger docs (if using swaggo)
swagger:
	swag init -g cmd/main.go

.PHONY: build clean test test-coverage deps run generate-module generate-full generate-help build-linux docker-build fmt lint install-deps migrate-up migrate-down swagger