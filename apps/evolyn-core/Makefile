.PHONY: base build run test test-integration vet clean swagger fmt lint init install-swagger postgres redis exec-db ui

GIT_VERSION = $(shell git describe --tags 2>/dev/null)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_STATE = $(shell test -z "$$(git status --porcelain 2>/dev/null)" && echo "clean" || echo "dirty")
BUILD_DATE = $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

BINARY = evolyn-core

LDFLAGS = -X evolyn/internal/version.gitVersion=$(GIT_VERSION) \
	-X evolyn/internal/version.gitCommit=$(GIT_COMMIT) \
	-X evolyn/internal/version.gitBranch=$(GIT_BRANCH) \
	-X evolyn/internal/version.gitTreeState=$(GIT_STATE) \
	-X evolyn/internal/version.buildDate=$(BUILD_DATE)

PKGS = $(shell go list ./...)
GOFILES = $(shell find . -name "*.go" -type f -not -path "./vendor/*")

# 本地 PostgreSQL：默认复用已存在的 postgres 容器（postgres:14-alpine），
# 5432 端口已有实例监听时直接复用，不再另起容器或拉新镜像
PG_CONTAINER ?= postgres
PG_IMAGE ?= postgres:14-alpine
PG_HOST ?= 127.0.0.1
PG_PORT ?= 5432
PG_PASSWORD ?= postgres
TEST_PG_DSN ?= postgres://postgres:$(PG_PASSWORD)@$(PG_HOST):$(PG_PORT)/postgres?sslmode=disable

base: clean test swagger fmt build

help: ## display the help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

build: ## build server binary to bin/
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/api

run: ## run server locally (needs config/app.yaml, postgres & redis)
	go run ./cmd/api

test: ## run unit test
	go test -ldflags -s -v -coverprofile=cover.out $(PKGS)
	go tool cover -func=cover.out -o coverage.txt

test-integration: ## run all tests incl. PostgreSQL integration (SEC-TENANT/MIGRATE-INT, FIX-022/023)
	@if nc -z $(PG_HOST) $(PG_PORT) 2>/dev/null; then \
		echo "reusing postgres at $(PG_HOST):$(PG_PORT)"; \
	else \
		echo "starting local postgres container $(PG_CONTAINER) ($(PG_IMAGE))"; \
		docker start $(PG_CONTAINER) 2>/dev/null || docker run --name $(PG_CONTAINER) -d -p $(PG_PORT):5432 -e POSTGRES_PASSWORD=$(PG_PASSWORD) $(PG_IMAGE); \
		until nc -z $(PG_HOST) $(PG_PORT) 2>/dev/null; do echo "wait postgres start"; sleep 1; done; \
	fi
	TEST_PG_DSN='$(TEST_PG_DSN)' go test ./...

vet: ## go vet
	go vet ./...

clean: ## clean bin and go mod
	@rm -rf bin/
	go mod tidy

fmt: ## golang format
	gofmt -s -w $(GOFILES)

lint: ## golangci-lint (requires golangci-lint installed)
	golangci-lint run

swagger: ## regenerate swagger docs into ./docs (gitignored, for local review only)
	swag init -g cmd/api/main.go -o docs --parseInternal --parseDependency --parseDepth 3

init: install-swagger postgres redis ## install all dependencies and init config
	echo "init all"

install-swagger: ## install swagger from golang (>= v1.16 required for aliased embedded models e.g. kernel.BaseModel)
	go install github.com/swaggo/swag/cmd/swag@v1.16.4

postgres: ## init local postgres and import scripts/db.sql
	@docker start $(PG_CONTAINER) 2>/dev/null || docker run --name $(PG_CONTAINER) -d -p $(PG_PORT):5432 -e POSTGRES_PASSWORD=$(PG_PASSWORD) $(PG_IMAGE)
	until nc -z $(PG_HOST) $(PG_PORT) 2>/dev/null; do echo "wait postgres start"; sleep 1; done
	cat scripts/db.sql | docker exec -i $(PG_CONTAINER) psql -U postgres

exec-db: ## exec to db container
	docker exec -it $(PG_CONTAINER) psql -d evolyn -U postgres

redis: ## init local redis (password matches config/app.yaml dev default)
	@docker start myredis || docker run --name myredis -d -p 6379:6379 redis --appendonly yes --requirepass 123456

ui: ## run frontend locally (sibling directory)
	cd ../evolyn-web && npm install && npm run dev
