##############################################################
# variables
##############################################################
SHELL                   := /bin/bash
SRC						?= $(shell pwd)
NAME					?= signoz
OS                      ?= $(shell uname -s | tr '[A-Z]' '[a-z]')
ARCH                    ?= $(shell uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')
COMMIT_SHORT_SHA        ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BRANCH_NAME             ?= $(subst /,-,$(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo local))
VERSION                 ?= $(BRANCH_NAME)-$(COMMIT_SHORT_SHA)
TIMESTAMP               ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
ARCHS					?= amd64 arm64
TARGET_DIR              ?= $(shell pwd)/target


GO_BUILD_VERSION_LDFLAGS 		= -X github.com/SigNoz/signoz/pkg/version.version=$(VERSION) -X github.com/SigNoz/signoz/pkg/version.hash=$(COMMIT_SHORT_SHA) -X github.com/SigNoz/signoz/pkg/version.time=$(TIMESTAMP) -X github.com/SigNoz/signoz/pkg/version.branch=$(BRANCH_NAME)
GO_BUILD_ARCHS_COMMUNITY 		= $(addprefix go-build-community-,$(ARCHS))
GO_BUILD_CONTEXT_COMMUNITY 		= $(SRC)/cmd/community
GO_BUILD_LDFLAGS_COMMUNITY 		= $(GO_BUILD_VERSION_LDFLAGS) -X github.com/SigNoz/signoz/pkg/version.variant=community

DOCKER_BUILD_ARCHS_COMMUNITY 	= $(addprefix docker-build-community-,$(ARCHS))
DOCKERFILE_COMMUNITY 			= $(SRC)/cmd/community/Dockerfile
DOCKER_REGISTRY_COMMUNITY 		?= ghcr.io/siginsight/siginsight
JS_BUILD_CONTEXT 				= $(SRC)/frontend

##############################################################
# directories
##############################################################
$(TARGET_DIR):
	mkdir -p $(TARGET_DIR)

##############################################################
# common commands
##############################################################
.PHONY: help
help: ## Displays help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-z0-9A-Z_-]+:.*?##/ { printf "  \033[36m%-40s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##############################################################
# devenv commands
##############################################################
.PHONY: devenv-clickhouse
devenv-clickhouse: ## Run clickhouse in devenv
	@cd .devenv/docker/clickhouse; \
	docker compose -f compose.yaml up -d

.PHONY: devenv-siginsight-otel-collector
devenv-siginsight-otel-collector: ## Run siginsight-otel-collector in devenv (requires clickhouse to be running)
	@cd .devenv/docker/siginsight-otel-collector; \
	docker compose -f compose.yaml up -d

.PHONY: devenv-up
devenv-up: devenv-clickhouse devenv-siginsight-otel-collector ## Start both clickhouse and siginsight-otel-collector for local development
	@echo "Development environment is ready!"
	@echo "   - ClickHouse: http://localhost:8123"
	@echo "   - SigInsight OTel Collector: grpc://localhost:4317, http://localhost:4318"

.PHONY: devenv-clickhouse-clean
devenv-clickhouse-clean: ## Clean all ClickHouse data from filesystem
	@echo "Removing ClickHouse data..."
	@rm -rf .devenv/docker/clickhouse/fs/tmp/*
	@echo "ClickHouse data cleaned!"

##############################################################
# go commands
##############################################################

.PHONY: go-test
go-test: ## Runs go unit tests
	@go test -race ./...

.PHONY: go-run-community
go-run-community: ## Runs the community go backend server (requires .env, see .env.example)
	@if [ ! -f .env ]; then \
		echo "Error: .env not found. Run: cp .env.example .env"; \
		exit 1; \
	fi
	@set -a; . ./.env; set +a; \
	go run -race \
		$(GO_BUILD_CONTEXT_COMMUNITY)/*.go server

.PHONY: go-build-community $(GO_BUILD_ARCHS_COMMUNITY)
go-build-community: ## Builds the go backend server for community
go-build-community: $(GO_BUILD_ARCHS_COMMUNITY)
$(GO_BUILD_ARCHS_COMMUNITY): go-build-community-%: $(TARGET_DIR)
	@mkdir -p $(TARGET_DIR)/$(OS)-$*
	@echo ">> building binary $(TARGET_DIR)/$(OS)-$*/$(NAME)-community"
	@if [ $* = "arm64" ]; then \
		CGO_ENABLED=0 GOARCH=$* GOOS=$(OS) go build -C $(GO_BUILD_CONTEXT_COMMUNITY) -tags timetzdata -o $(TARGET_DIR)/$(OS)-$*/$(NAME)-community -ldflags "-s -w $(GO_BUILD_LDFLAGS_COMMUNITY)"; \
	else \
		CGO_ENABLED=0 GOARCH=$* GOOS=$(OS) go build -C $(GO_BUILD_CONTEXT_COMMUNITY) -tags timetzdata -o $(TARGET_DIR)/$(OS)-$*/$(NAME)-community -ldflags "-s -w $(GO_BUILD_LDFLAGS_COMMUNITY)"; \
	fi



##############################################################
# js commands
##############################################################
.PHONY: js-build
js-build: ## Builds the js frontend
	@echo ">> building js frontend"
	@cd $(JS_BUILD_CONTEXT) && CI=1 yarn install && yarn build

##############################################################
# python integration commands
##############################################################
.PHONY: py-fmt
py-fmt: ## Formats integration python tests
	@cd tests/integration && uv run autoflake --in-place --remove-all-unused-imports --remove-unused-variables --recursive .
	@cd tests/integration && uv run isort .
	@cd tests/integration && uv run black .

.PHONY: py-lint
py-lint: ## Lints integration python tests
	@cd tests/integration && uv run pylint .

##############################################################
# docker commands
##############################################################
.PHONY: docker-build-community $(DOCKER_BUILD_ARCHS_COMMUNITY)
docker-build-community: ## Builds the docker image for community
docker-build-community: $(DOCKER_BUILD_ARCHS_COMMUNITY)
$(DOCKER_BUILD_ARCHS_COMMUNITY): docker-build-community-%: go-build-community-% js-build
	@echo ">> building docker image for $(NAME)-community"
	@docker build -t "$(DOCKER_REGISTRY_COMMUNITY):$(VERSION)-$*" \
		--build-arg TARGETARCH="$*" \
		-f $(DOCKERFILE_COMMUNITY) $(SRC)

.PHONY: docker-buildx-community
docker-buildx-community: ## Builds the docker image for community using buildx
docker-buildx-community: go-build-community js-build
	@echo ">> building docker image for $(NAME)-community"
	@docker buildx build --file $(DOCKERFILE_COMMUNITY) \
		--progress plain \
		--platform linux/arm64,linux/amd64 \
		--push \
		--tag $(DOCKER_REGISTRY_COMMUNITY):$(VERSION) $(SRC)


##############################################################
# generate commands
##############################################################
.PHONY: gen-mocks
gen-mocks:
	@echo ">> Generating mocks"
	@mockery --config .mockery.yml
