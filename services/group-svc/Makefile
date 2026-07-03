BINARY=group-svc
BUILD_DIR=./build
CMD_DIR=./cmd
MAIN_PATH=$(CMD_DIR)/main.go
COMMIT_HASH=$(shell git log --format="%h" -1 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)

.PHONY: all build run clean test lint

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.commitHash=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME)" -o $(BUILD_DIR)/$(BINARY) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./... -v -count=1 -race

test-cover:
	go test ./... -v -count=1 -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...
