.PHONY: all build clean test lint vet fmt tidy pre-commit run-gateway run-upstream

all: build

build: gateway upstream

gateway:
	go build -o gateway ./cmd/gateway

upstream:
	go build -o upstream ./cmd/upstream

clean:
	rm -f gateway upstream

test:
	go test ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

pre-commit: tidy fmt vet lint

run-gateway: gateway
	./gateway

run-upstream: upstream
	./upstream
