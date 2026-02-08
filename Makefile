VERSION ?= dev

.PHONY: test build build-linux

test:
	go test ./...

build:
	mkdir -p bin
	go build -ldflags "-X main.Version=$(VERSION)" -o bin/mightymonitor-agent ./cmd/mightymonitor-agent

build-linux:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.Version=$(VERSION)" -o dist/mightymonitor-agent-linux-amd64 ./cmd/mightymonitor-agent
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.Version=$(VERSION)" -o dist/mightymonitor-agent-linux-arm64 ./cmd/mightymonitor-agent

