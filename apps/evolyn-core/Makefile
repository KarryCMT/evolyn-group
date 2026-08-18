.PHONY: base build run test clean swagger fmt lint init install-swagger postgres redis exec-db ui

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
	@docker start mypostgres || docker run --name mypostgres -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres
	until docker exec mypostgres psql -U postgres; do echo "wait postgres start"; sleep 1; done
	cat scripts/db.sql | docker exec -i mypostgres psql -U postgres

exec-db: ## exec to db container
	docker exec -it mypostgres psql -d evolyn -U postgres

redis: ## init local redis (password matches config/app.yaml dev default)
	@docker start myredis || docker run --name myredis -d -p 6379:6379 redis --appendonly yes --requirepass 123456

ui: ## run frontend locally (sibling directory)
	cd ../evolyn-web && npm install && npm run dev
