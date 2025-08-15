all: build test lint format

build:
	go build .

test:
	go test ./...

lint:
	golangci-lint run

format:
	golangci-lint fmt