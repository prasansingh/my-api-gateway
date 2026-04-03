.PHONY: all build build-linux clean test lint vet fmt tidy pre-commit run-gateway run-upstream loadgen

all: build

build: gateway upstream loadgen

gateway:
	go build -o gateway ./cmd/gateway

upstream:
	go build -o upstream ./cmd/upstream

loadgen:
	go build -o loadgen ./cmd/loadgen

build-linux:
	GOOS=linux GOARCH=amd64 go build -o gateway ./cmd/gateway

clean:
	rm -f gateway upstream loadgen

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
