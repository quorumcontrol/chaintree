FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: build

lint: $(FIRSTGOPATH)/bin/golangci-lint
	$(FIRSTGOPATH)/bin/golangci-lint run

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

test: $(gosources) go.mod go.sum
	go test ./... -tags=integration

build: $(gosources) go.mod go.sum
	go build ./...

clean:
	go clean ./...

.PHONY: all build test clean lint
