.PHONY: all build build-linux clean test lint vet fmt tidy pre-commit run-gateway run-upstream loadgen deploy

EC2_HOST := ec2-user@ec2-32-192-103-90.compute-1.amazonaws.com
EC2_KEY  := jit-server.pem

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
	GOOS=linux GOARCH=amd64 go build -o upstream ./cmd/upstream
	GOOS=linux GOARCH=amd64 go build -o loadgen ./cmd/loadgen

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

deploy: build-linux
	scp -i $(EC2_KEY) gateway upstream loadgen config.yaml $(EC2_HOST):/tmp/
	scp -i $(EC2_KEY) ec2-services/*.service $(EC2_HOST):/tmp/
	ssh -i $(EC2_KEY) $(EC2_HOST) '\
		sudo mv /tmp/gateway /tmp/upstream /tmp/loadgen /usr/local/bin/ && \
		sudo mkdir -p /etc/my-api-gateway && \
		sudo mv /tmp/config.yaml /etc/my-api-gateway/ && \
		sudo mv /tmp/*.service /etc/systemd/system/ && \
		sudo systemctl daemon-reload && \
		sudo systemctl restart my-api-gateway upstream'

run-gateway: gateway
	./gateway

run-upstream: upstream
	./upstream
