GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell awk 'NR==2 {print $0}' .project)

ATLAS_VERSION ?= v1.1.0
ATLAS_BIN := $(CURDIR)/bin/atlas$(if $(filter windows,$(GOHOSTOS)),.exe,)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
endif

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run unit tests.
	go test ./...

.PHONY: test-sync-kernel
test-sync-kernel: ## Run container-based sync kernel tests (ChangeStore + gRPC two-node).
	go test -v -count=1 ./internal/data/repository/changedag
	go test -v -count=1 ./internal/service/sync


.PHONY: check-commit
check-commit: ## Run strict local commit gate checks.
	@echo "Running commit gate checks..."
	$(MAKE) fmt
	$(MAKE) vet
	$(MAKE) all
	$(MAKE) build
	go test ./...
	$(MAKE) test-sync-kernel
	@echo "Commit gate checks passed."

.PHONY: install-githooks
install-githooks: ## Configure repository hooks and install pre-commit gate.
	@chmod +x .githooks/pre-commit
	@git config core.hooksPath .githooks
	@echo "Installed git hooks at .githooks (core.hooksPath)."


.PHONY: infra-up
infra-up: ## Start local infra (postgres/redis) via docker compose
	@docker compose -p projecttemplate-infra --project-directory docker -f docker/docker-compose.yml --profile local up -d postgres redis

.PHONY: infra-down
infra-down: ## Stop local infra (postgres/redis) via docker compose
	@docker compose -p projecttemplate-infra --project-directory docker -f docker/docker-compose.yml --profile local down

.PHONY: infra-ps
infra-ps: ## Show infra containers status
	@docker compose -p projecttemplate-infra --project-directory docker -f docker/docker-compose.yml --profile local ps

.PHONY: infra-logs
infra-logs: ## Tail infra logs
	@docker compose -p projecttemplate-infra --project-directory docker -f docker/docker-compose.yml --profile local logs -f --tail=200

.PHONY: install-protoc
# install protoc for multiple platforms (Linux x86_64/aarch64, macOS x86_64/ARM64)
install-protoc:
	@echo "Installing protoc for current platform..."
	@if command -v protoc >/dev/null 2>&1; then \
		echo "protoc is already installed: $$(protoc --version)"; \
	else \
		OS=$$(uname -s); \
		ARCH=$$(uname -m); \
		PROTOC_VERSION="31.1"; \
		case "$$OS" in \
			"Darwin") \
				if [ "$$ARCH" = "arm64" ]; then \
					PROTOC_PLATFORM="osx-aarch_64"; \
				else \
					PROTOC_PLATFORM="osx-x86_64"; \
				fi \
				;; \
			"Linux") \
				if [ "$$ARCH" = "aarch64" ] || [ "$$ARCH" = "arm64" ]; then \
					PROTOC_PLATFORM="linux-aarch_64"; \
				else \
					PROTOC_PLATFORM="linux-x86_64"; \
				fi \
				;; \
			*) \
				echo "Unsupported platform: $$OS/$$ARCH"; \
				echo "Please install protoc manually from https://github.com/protocolbuffers/protobuf/releases"; \
				exit 1; \
				;; \
		esac; \
		PROTOC_ZIP="protoc-$$PROTOC_VERSION-$$PROTOC_PLATFORM.zip"; \
		echo "Detected platform: $$OS/$$ARCH -> $$PROTOC_PLATFORM"; \
		echo "Downloading $$PROTOC_ZIP..."; \
		curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v$$PROTOC_VERSION/$$PROTOC_ZIP"; \
		if [ $$? -ne 0 ]; then \
			echo "Failed to download protoc. Please check your internet connection."; \
			exit 1; \
		fi; \
		echo "Extracting protoc to protocbuf-release..."; \
		mkdir -p protocbuf-release; \
		unzip -q "$$PROTOC_ZIP" -d protocbuf-release; \
		if [ ! -f "protocbuf-release/bin/protoc" ]; then \
			echo "protoc binary not found in archive. Installation failed."; \
			rm -rf protocbuf-release "$$PROTOC_ZIP"; \
			exit 1; \
		fi; \
		echo "Installing protoc to /usr/local/bin/..."; \
		sudo cp protocbuf-release/bin/protoc /usr/local/bin/; \
		sudo chmod +x /usr/local/bin/protoc; \
		echo "Cleaning up temporary files..."; \
		rm -rf protocbuf-release "$$PROTOC_ZIP"; \
		echo "protoc installed successfully: $$(protoc --version)"; \
	fi

.PHONY: show-platform
# show current platform information
show-platform:
	@echo "=== Platform Information ==="
	@OS=$$(uname -s); \
	ARCH=$$(uname -m); \
	echo "Operating System: $$OS"; \
	echo "Architecture: $$ARCH"; \
	case "$$OS" in \
		"Darwin") \
			if [ "$$ARCH" = "arm64" ]; then \
				echo "Platform: macOS ARM64 (Apple Silicon)"; \
				echo "Protoc Package: protoc-31.1-osx-aarch_64.zip"; \
			else \
				echo "Platform: macOS x86_64 (Intel)"; \
				echo "Protoc Package: protoc-31.1-osx-x86_64.zip"; \
			fi \
			;; \
		"Linux") \
			if [ "$$ARCH" = "aarch64" ] || [ "$$ARCH" = "arm64" ]; then \
				echo "Platform: Linux ARM64"; \
				echo "Protoc Package: protoc-31.1-linux-aarch_64.zip"; \
			else \
				echo "Platform: Linux x86_64"; \
				echo "Protoc Package: protoc-31.1-linux-x86_64.zip"; \
			fi \
			;; \
		*) \
			echo "Platform: $$OS/$$ARCH (Unsupported)"; \
			echo "Please install protoc manually"; \
			;; \
	esac
	@echo ""

.PHONY: show-tools
# show installed tools versions
show-tools:
	@echo "=== Development Tools Status ==="
	@echo "protoc: $$(protoc --version 2>/dev/null || echo 'Not installed')"
	@echo "atlas(system): $$(atlas version 2>/dev/null || echo 'Not installed')"
	@echo "atlas(repo): $$(test -x \"$(ATLAS_BIN)\" && \"$(ATLAS_BIN)\" version || echo 'Not installed')"
	@echo ""
	@echo "=== Go Tools (via go.mod tool directive) ==="
	@go list tool 2>/dev/null || echo "No tools found in go.mod"
	@echo ""
	@echo "=== System-wide Go Tools ==="
	@echo "kratos: $$(kratos --version 2>/dev/null || echo 'Not installed globally')"
	@echo "protoc-gen-go: $$(protoc-gen-go --version 2>/dev/null || echo 'Not installed globally')"
	@echo "protoc-gen-validate: $$(which protoc-gen-validate >/dev/null 2>&1 && echo 'Installed globally' || echo 'Not installed globally')"
	@echo ""
	@echo "=== Usage ==="
	@echo "Go 1.24+ tools: go tool <n> (e.g., go tool protoc-gen-go --version)"
	@echo "Global tools: <tool-name> (e.g., kratos --version)"
	@echo ""
	@echo "=== Tool Dependencies in go.mod ==="
	@go list -m all | grep -E "(protoc-gen|kratos|gnostic)" | sort

.PHONY: ensure-atlas
ensure-atlas: ## Ensure Atlas CLI is installed (required for migrations)
	@mkdir -p "$(CURDIR)/bin"
	@echo "Ensuring atlas $(ATLAS_VERSION) in $(ATLAS_BIN)"
	@if [ -x "$(ATLAS_BIN)" ] && "$(ATLAS_BIN)" version 2>/dev/null | grep -F "atlas version $(ATLAS_VERSION)" >/dev/null 2>&1; then \
		echo "atlas $(ATLAS_VERSION) already present"; \
	else \
		ATLAS_VERSION="$(ATLAS_VERSION)" sh -c "$$(curl -fsSL https://atlasgo.sh)" -- --no-install -y -o "$(ATLAS_BIN)" >/dev/null; \
		chmod +x "$(ATLAS_BIN)"; \
	fi
	@"$(ATLAS_BIN)" version >/dev/null

.PHONY: install-tools-global
# install tools globally to system (accessible without 'go tool' prefix) with fixed versions
install-tools-global: install-protoc
	@echo "Installing Go tools globally to system with fixed versions..."
	@echo "Tools will be available directly as commands (e.g., 'kratos', 'protoc-gen-go')"
	@echo ""
	@echo "Installing tools with fixed versions..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0
	go install github.com/envoyproxy/protoc-gen-validate@v1.2.1
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@echo ""
	@echo "All tools installed globally with fixed versions! Tools are now available in your PATH."
	@echo "Installation path: $$(go env GOPATH)/bin or $$(go env GOBIN)"

.PHONY: install-tools
# install tools using Go 1.24+ tool directive with fixed versions (recommended)
install-tools: install-protoc
	@echo "Installing Go tools using Go 1.24+ tool directive with fixed versions..."
	@echo "Adding tools to go.mod..."
	go get -tool google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	go get -tool google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go get -tool github.com/go-kratos/kratos/cmd/kratos/v2@v2.9.1
	go get -tool github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.9.1
	go get -tool github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0
	go get -tool github.com/envoyproxy/protoc-gen-validate@v1.2.1
	@echo "All tools installed successfully with fixed versions! Use 'go tool <n>' to run them."

.PHONY: init
# init env (with fixed versions)
init: install-protoc
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
	go install github.com/go-kratos/kratos/cmd/kratos/v2@v2.9.1
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.9.1
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@v0.7.0
	go install github.com/envoyproxy/protoc-gen-validate@v1.2.1

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	@echo "Sorting proto fields..."
	go run hack/script/sort_proto_fields.go -path ./api -recursive -fill-gaps
	@echo "Generating proto code..."
	protoc --proto_path=./api \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./api \
 	       --go-http_out=paths=source_relative:./api \
 	       --go-grpc_out=paths=source_relative:./api \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
		   --experimental_allow_proto3_optional \
	       $(API_PROTO_FILES)
	make validate

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -trimpath -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

PHONY: error
error :
	protoc --proto_path=. \
	--proto_path=./third_party \
	--go_out=paths=source_relative:. \
	--go-errors_out=paths=source_relative:. \
		 api/common/v1/error.proto

PHONY: run
run:
	go run cmd/server/main.go --conf docker/backend/config/config.yaml

.PHONY: dev-bootstrap
dev-bootstrap: ## Local dev: start infra + apply migrations + seed minimal data (non-destructive)
	@$(MAKE) infra-up
	@$(MAKE) migrate-dev
	@$(MAKE) seed-dev

.PHONY: dev-run
dev-run: ## Local dev: bootstrap (infra+migrate+seed) then run backend (OpenAPI enabled)
	@$(MAKE) dev-bootstrap
	@RST_DEV_ENABLE_OPENAPI=true RST_DEV_OPENAPI_PATH=./openapi.yaml $(MAKE) run

.PHONY: seed-dev
seed-dev: ## Seed minimal local dev data (admin user/role) into DATABASE_URL (default: local docker infra)
	go run hack/seed/main.go

# Database migration - 智能兼容多种情况
# 开发环境（自动检测）：make migrate-dev
# 生产环境（安全迁移）：make migrate-prod
# 生成迁移文件：      make migrate-diff NAME=xxx
# 查看状态：          make migrate-status
# 重置数据库：        make migrate-reset

.PHONY: migrate-dev
migrate-dev: ensure-atlas ## 开发环境：智能检测数据库状态，自动选择同步方式
	@go run hack/dbmigrate/main.go dev

.PHONY: migrate-prod
migrate-prod: ensure-atlas ## 生产环境：使用 Atlas 安全应用迁移
	@go run hack/dbmigrate/main.go prod

.PHONY: migrate-diff
migrate-diff: ensure-atlas ## 生成迁移文件 (usage: make migrate-diff NAME=add_user_phone)
	@go run hack/dbmigrate/main.go diff $(NAME)

.PHONY: migrate-status
migrate-status: ensure-atlas ## 查看迁移状态
	@go run hack/dbmigrate/main.go status

.PHONY: migrate-reset
migrate-reset: ensure-atlas ## 清空数据库并重新初始化（危险！）
	@go run hack/dbmigrate/main.go reset

# 传统 Atlas 命令（直接使用）
.PHONY: migrate-apply
migrate-apply: ensure-atlas ## 直接应用 Atlas 迁移
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "错误: DATABASE_URL 不能为空"; \
		echo "示例: export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'"; \
		exit 1; \
	fi
	cd migrations && DATABASE_URL="$(DATABASE_URL)" "$(ATLAS_BIN)" migrate apply --env local

.PHONY: migrate-lint
migrate-lint: ensure-atlas ## Lint 迁移文件
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "错误: DATABASE_URL 不能为空"; \
		echo "示例: export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'"; \
		exit 1; \
	fi
	cd migrations && DATABASE_URL="$(DATABASE_URL)" "$(ATLAS_BIN)" migrate lint --env local

.PHONY: migrate-inspect
migrate-inspect: ensure-atlas ## 查看数据库当前 schema
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "错误: DATABASE_URL 不能为空"; \
		echo "示例: export DATABASE_URL='postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable'"; \
		exit 1; \
	fi
	cd migrations && DATABASE_URL="$(DATABASE_URL)" "$(ATLAS_BIN)" schema inspect --env local

# generate validate proto
.PHONY: validate
validate:
	protoc --proto_path=. \
			--proto_path=./third_party \
			--go_out=paths=source_relative:. \
			--validate_out=paths=source_relative,lang=go:. \
			--experimental_allow_proto3_optional \
			$(API_PROTO_FILES)

PHONY: buildall
buildall:fmt vet ## Build manager binary.
	$(eval PRODUCT_NAME := $(shell awk 'NR==1 {print $$0}' .project))
	$(eval VERSION := $(shell awk 'NR==2 {print $$0}' .project))
	@echo "Building $(PRODUCT_NAME) version $(VERSION) for $(ENV) environment"
	pushd cmd/server && \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-X main.Version=$(VERSION)" -o ../../dist/backend/$(PRODUCT_NAME) &&\
	popd
	cd web && pnpm i && pnpm build:notypecheck && mkdir -p ../dist/frontend && cp -r dist/* ../dist/frontend/ && cd ..
	@echo "Building Docker images (unified docker/ layout)..."
	DOCKER_BUILDKIT=1 docker build -f docker/backend/Dockerfile -t $(PRODUCT_NAME)-backend:$(VERSION) -t $(PRODUCT_NAME)-backend:latest .
	DOCKER_BUILDKIT=1 docker build -f docker/frontend/Dockerfile -t $(PRODUCT_NAME)-frontend:$(VERSION) -t $(PRODUCT_NAME)-frontend:latest .

.PHONY: docker-build-prod
docker-build-prod: ## Build Docker package for production environment
	@echo "Building Docker package for production environment..."
	rm -rf dist/
	ENV=prod make buildall
	./hack/script/build-docker.sh prod

.PHONY: docker-build-dev
docker-build-dev: ## Build Docker package for development environment
	@echo "Building Docker package for development environment..."
	rm -rf dist/
	ENV=dev make buildall
	./hack/script/build-docker.sh dev

.PHONY: docker-build
docker-build: ## Build Docker package (requires environment parameter: make docker-build ENV=prod or make docker-build ENV=dev)
	@if [ -z "$(ENV)" ]; then \
		echo "错误: 必须指定环境参数"; \
		echo "用法: make docker-build ENV=prod 或 make docker-build ENV=dev"; \
		exit 1; \
	fi
	@if [ "$(ENV)" != "prod" ] && [ "$(ENV)" != "dev" ]; then \
		echo "错误: 环境参数必须是 'prod' 或 'dev'"; \
		echo "用法: make docker-build ENV=prod 或 make docker-build ENV=dev"; \
		exit 1; \
	fi
	@echo "Building Docker package for $(ENV) environment..."
	ENV=$(ENV) make buildall
	./hack/script/build-docker.sh $(ENV)

.PHONY: docker-clean
docker-clean: ## Clean Docker build artifacts
	@echo "Cleaning Docker build artifacts..."
	rm -rf dist/
	@echo "Docker build artifacts cleaned."

.PHONY: docker-help
docker-help: ## Show Docker build help
	@echo "=== Docker Build Commands ==="
	@echo "make docker-build-prod    - Build production Docker package"
	@echo "make docker-build-dev     - Build development Docker package"
	@echo "make docker-build ENV=prod - Build production Docker package (alternative)"
	@echo "make docker-build ENV=dev  - Build development Docker package (alternative)"
	@echo "make docker-clean         - Clean Docker build artifacts"
	@echo ""
	@echo "=== Environment Differences ==="
	@echo "prod: Production environment with optimized settings"
	@echo "dev:  Development environment with debug ports and verbose logging"
