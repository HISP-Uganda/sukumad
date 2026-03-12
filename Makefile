# Project variables
APP_NAME = sukumad
CMD_DIR  = ./cmd/api
MIGRATIONS_DIR = ./db/migrations
DB_URL = $(DATABASE_URI)
PORT ?= 8383

# Tools
SWAG   = swag
MIGRATE = migrate

# Default target
.PHONY: all
all: run

## ---- Development ----

.PHONY: run
run: ## Run the application
	go run $(CMD_DIR)

.PHONY: watch
watch: ## Run the application with live reload (requires air)
	air -c .air.toml

.PHONY: build
build: ## Build the binary
	go build -o bin/$(APP_NAME) $(CMD_DIR)

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

## ---- Swagger / OpenAPI ----

.PHONY: swag
swag: ## Generate Swagger documentation
	$(SWAG) init -g $(CMD_DIR)/main.go -o ./internal/docs

## ---- Database ----

.PHONY: migrate-up
migrate-up: ## Apply all up migrations
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

.PHONY: migrate-down
migrate-down: ## Rollback all migrations
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down

.PHONY: migrate-force
migrate-force: ## Force set database migration version
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DB_URL)" force $(version)

.PHONY: migrate-new
migrate-new: ## Create a new migration (usage: make migrate-new name=add_users)
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

## ---- Helpers ----

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
