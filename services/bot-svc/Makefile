.PHONY: build run test clean

build:
	go build -o bin/bot-svc ./cmd/

run:
	go run ./cmd/ --config config.yaml

test:
	go test ./... -v -race -count=1

clean:
	rm -rf bin/

docker-build:
	docker build -t shulian/bot-svc:latest -f Dockerfile ../../
