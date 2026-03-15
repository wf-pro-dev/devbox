.PHONY: build build-cli build-all build-web generate run run-web test clean tidy install docker-build docker-up docker-down docker-logs-backend docker-logs-ui docker-restart docker-deploy

BUILD_TAGS := -tags fts5
DOCKER_COMPOSE_FILE := docker/docker-compose.yml

## Generate sqlc Go code
generate:
	sqlc generate

## Build the SvelteKit frontend
build-web:
	cd web && npm install && npm run build

## Build the server binary (requires frontend to be built first)
build: generate build-web
	go build $(BUILD_TAGS) -o bin/devbox ./cmd/devbox

## Build the CLI binary
build-cli:
	go build $(BUILD_TAGS) -o bin/devbox-cli ./cmd/devbox-cli

## Full production build: frontend + server + CLI
build-all: build build-cli

## Install CLI to ~/.local/bin
install: build-cli
	mkdir -p $(HOME)/.local/bin
	cp bin/devbox-cli $(HOME)/.local/bin/devbox-cli
	@echo "✓ installed to $(HOME)/.local/bin/devbox-cli"

## Run the Go backend only (for dev with vite dev server)
run: generate
	export $$(grep -v '^#' .env | xargs) && go run $(BUILD_TAGS) ./cmd/devbox

## Run the SvelteKit dev server (proxies API to Go backend on :8888)
run-web:
	cd web && npm run dev

## Run tests
test:
	go test $(BUILD_TAGS) ./...

## Tidy Go dependencies
tidy:
	go mod tidy

## Remove built artifacts
clean:
	rm -rf bin/ web/build/ web/node_modules/.vite

## ── Docker ────────────────────────────────────────────────────────────────

## Build the Docker image
docker-build:
	docker compose -f $(DOCKER_COMPOSE_FILE) build

## Start devbox in the background
docker-up:
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d

## Stop devbox
docker-down:
	docker compose -f $(DOCKER_COMPOSE_FILE) down

## Tail logs
docker-logs-backend:
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f devbox-backend

docker-logs-ui:
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f devbox-ui

## Restart the container (e.g. after a config change)
docker-restart:
	docker compose -f $(DOCKER_COMPOSE_FILE) restart devbox-backend
	docker compose -f $(DOCKER_COMPOSE_FILE) restart devbox-ui

## Rebuild image and restart (full redeploy)
docker-deploy:
	docker compose -f $(DOCKER_COMPOSE_FILE) build && docker compose -f $(DOCKER_COMPOSE_FILE) up -d