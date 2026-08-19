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

base: clean test swagger fmt build ## 本地完整验收链：清理 → 单测 → swagger → 格式化 → 构建

help: ## 显示所有可用目标及说明
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

build: ## 构建服务二进制到 bin/（ldflags 注入版本信息）
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/api

run: ## 本地运行服务（需 config/app.yaml，postgres/redis 就绪；启动时应用版本化迁移）
	go run ./cmd/api

test: ## 单元测试（含覆盖率，产物 cover.out/coverage.txt）
	go test -ldflags -s -v -coverprofile=cover.out $(PKGS)
	go tool cover -func=cover.out -o coverage.txt

test-integration: ## 全量测试（复用本地 postgres 跑集成用例 SEC-TENANT/MIGRATE-INT，FIX-022/023）
	@if nc -z $(PG_HOST) $(PG_PORT) 2>/dev/null; then \
		echo "reusing postgres at $(PG_HOST):$(PG_PORT)"; \
	else \
		echo "starting local postgres container $(PG_CONTAINER) ($(PG_IMAGE))"; \
		docker start $(PG_CONTAINER) 2>/dev/null || docker run --name $(PG_CONTAINER) -d -p $(PG_PORT):5432 -e POSTGRES_PASSWORD=$(PG_PASSWORD) $(PG_IMAGE); \
		until nc -z $(PG_HOST) $(PG_PORT) 2>/dev/null; do echo "wait postgres start"; sleep 1; done; \
	fi
	TEST_PG_DSN='$(TEST_PG_DSN)' go test ./...

vet: ## go vet 静态检查
	go vet ./...

clean: ## 清理 bin/ 并执行 go mod tidy
	@rm -rf bin/
	go mod tidy

fmt: ## gofmt -s 格式化全部 Go 文件
	gofmt -s -w $(GOFILES)

lint: ## golangci-lint 检查（需已安装 golangci-lint）
	golangci-lint run

swagger: ## 重新生成 swagger 文档到 ./docs（gitignored，仅本地查阅）
	swag init -g cmd/api/main.go -o docs --parseInternal --parseDependency --parseDepth 3

init: install-swagger postgres redis ## 一键初始化本地依赖（swag 工具 + postgres + redis）
	echo "init all"

install-swagger: ## 安装 swag CLI（需 >= v1.16：支持别名内嵌 model，如 kernel.BaseModel）
	go install github.com/swaggo/swag/cmd/swag@v1.16.4

postgres: ## 启动/复用本地 postgres 容器并导入 scripts/db.sql 终态快照
	@docker start $(PG_CONTAINER) 2>/dev/null || docker run --name $(PG_CONTAINER) -d -p $(PG_PORT):5432 -e POSTGRES_PASSWORD=$(PG_PASSWORD) $(PG_IMAGE)
	until nc -z $(PG_HOST) $(PG_PORT) 2>/dev/null; do echo "wait postgres start"; sleep 1; done
	cat scripts/db.sql | docker exec -i $(PG_CONTAINER) psql -U postgres

exec-db: ## 进入业务库 evolyn 的 psql 交互终端
	docker exec -it $(PG_CONTAINER) psql -d evolyn -U postgres

redis: ## 启动本地 redis（密码与 config/app.yaml 开发默认值一致）
	@docker start myredis || docker run --name myredis -d -p 6379:6379 redis --appendonly yes --requirepass 123456

ui: ## 本地运行前端（相邻目录 evolyn-web）
	cd ../evolyn-web && npm install && npm run dev
