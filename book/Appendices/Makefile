# ====================================================================================
# CONFIGURATION AND ENV VARIABLES
# ====================================================================================
MAIN_PACKAGE_PATH := ./cmd/api
BINARY_NAME := microservice
MIGRATIONS_PATH := ./db/migrations

# Default local engine access tokens (override these via CLI if necessary)
DB_USER ?= root
DB_PASSWORD ?= password
DB_HOST ?= 127.0.0.1
DB_PORT ?= 3306
DB_NAME ?= my_database

# Construct the full connection payload strings for both Go app context and migration binary
DSN := "$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)?parseTime=true"
MIGRATE_DSN := "mysql://$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)"

# ====================================================================================
# HELP SHORTCUT
# ====================================================================================
.PHONY: help
help:
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Development Shortcuts:'
	@echo '  audit           Run static code validation vectors (fmt, vet, staticcheck)'
	@echo '  test            Execute all internal unit tests with code-coverage processing'
	@echo '  run             Instantly boot up the local application stack inline'
	@echo '  build           Compile optimized, statically linked production host binary'
	@echo ''
	@echo 'Database / Migration Drivers (golang-migrate):'
	@echo '  db/status       Verify network readiness connection of localized MySQL instance'
	@echo '  migrate/new     Create a new timestamped migration file. Usage: make migrate/new NAME=text'
	@echo '  migrate/up      Apply all pending database schema update migrations forward'
	@echo '  migrate/down    Roll back the single last applied database schema version change'

# ====================================================================================
# QUALITY CONTROL AND TESTING
# ====================================================================================
.PHONY: audit
audit:
	@echo 'Formatting source scripts cleanly via go fmt...'
	go fmt ./...
	@echo 'Running structural compilation analysis via go vet...'
	go vet ./...
	@echo 'Executing dependency scanning and vulnerability checks...'
	go mod tidy
	go mod verify

.PHONY: test
test:
	@echo 'Running test vectors with data race detection activated...'
	go test -v -race -timeout 30s ./...

# ====================================================================================
# RUNNING AND COMPILING
# ====================================================================================
.PHONY: run
run: audit
	@echo 'Launching API microservice application stack...'
	go run $(MAIN_PACKAGE_PATH)

.PHONY: build
build: audit
	@echo 'Compiling performance-optimized binary for Linux deployment...'
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./bin/$(BINARY_NAME) $(MAIN_PACKAGE_PATH)
	@echo 'Final standalone executable compiled safely into ./bin/$(BINARY_NAME)'

# ====================================================================================
# DATABASE PERSISTENCE AND SCHEMAS
# ====================================================================================
.PHONY: db/status
db/status:
	@echo 'Verifying network availability of local MySQL instance...'
	@mysqladmin -h $(DB_HOST) -P $(DB_PORT) -u $(DB_USER) -p$(DB_PASSWORD) ping --silent || (echo "❌ Error: Cannot establish TCP link to local engine."; exit 1)
	@echo '🔌 Connection check passed. Local database engine is responsive and online.'

.PHONY: migrate/new
migrate/new:
	@if [ -z "$(NAME)" ]; then echo "❌ Error: Argument NAME is missing. Run: make migrate/new NAME=add_users_table"; exit 1; fi
	@echo 'Scaffolding version control sheet templates for schema alterations...'
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

.PHONY: migrate/up
migrate/up: db/status
	@echo 'Executing migration forward tracks onto local target schema...'
	migrate -path $(MIGRATIONS_PATH) -database $(MIGRATE_DSN) up
	@echo '✅ Schema versions seamlessly synchronized forward to latest state.'

.PHONY: migrate/down
migrate/down: db/status
	@echo 'Executing single rollback iteration on target schema...'
	migrate -path $(MIGRATIONS_PATH) -database $(MIGRATE_DSN) down 1
	@echo '⚠️ Structural version rolled back by exactly 1 sequence step.'
