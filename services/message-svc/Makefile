.PHONY: build run test clean

build:
	go build -o bin/message-svc ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test ./... -v -count=1

test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

lint:
	go vet ./...

clean:
	rm -rf bin/ coverage.out coverage.html
