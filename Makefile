.DEFAULT_GOAL = run

TYPESPEC_DIR="./docs/typespec"
LOCAL_CONFIG_PATH=".env.local"

BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LD_FLAGS=-X main.buildTime=$(BUILD_TIME)
BINARY_NAME=gopher

# Runs all services in detached mode.
.PHONY: run
run:
	docker compose --env-file .env.local up -d --build

# Runs all services without detached mode (for debugging).
.PHONY: rund
rund:
	docker compose --env-file .env.local up --build

# Start go app locally.
.PHONY: run-local
run-local:
	CONFIG_PATH=$(LOCAL_CONFIG_PATH) go run cmd/app/main.go

# Build go app.
.PHONY: build
build:
	go build -ldflags="$(LD_FLAGS)" -o $(BINARY_NAME) cmd/app/main.go

# Shows all service statuses.
.PHONY: services
services:
	docker compose ps

# Stops all running services.
.PHONY: stop
stop:
	docker compose down

# Cleans up all resources including volumes.
.PHONY: clean
clean:
	docker compose down -v

# Lint project.
.PHONY: lint
lint:
	golangci-lint run

# Install typespec libs.
.PHONY: tsp-install
tsp-install:
	cd $(TYPESPEC_DIR) && tsp install

# Compile typespec in openapi.
.PHONY: tsp-compile
tsp-compile: tsp-install
	cd $(TYPESPEC_DIR) && tsp compile . --option "@typespec/openapi3.emitter-output-dir={project-root}/"

# Generate http endpoints using generated openapi.
.PHONY: ogen-gen
ogen-gen: tsp-compile
	go run github.com/ogen-go/ogen/cmd/ogen@latest --target internal/router/ogen/ --clean docs/openapi.yaml

# Generate http endpoints.
.PHONY: code-gen
code-gen: tsp-compile ogen-gen

# Run unit tests.
.PHONY: test
test:
	go test --race --count=1 ./...

# Run benchmarks.
.PHONY: bench
bench:
	go test -bench=. -benchmem -count=1 -v ./...