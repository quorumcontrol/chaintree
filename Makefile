FIRSTGOPATH = $(firstword $(subst :, ,$(GOPATH)))

gosources = $(shell find . -path "./vendor/*" -prune -o -type f -name "*.go" -print)

all: build

lint: $(FIRSTGOPATH)/bin/golangci-lint
	$(FIRSTGOPATH)/bin/golangci-lint run

$(FIRSTGOPATH)/bin/golangci-lint:
	./scripts/download-golangci-lint.sh

vendor: Gopkg.toml Gopkg.lock
	dep ensure

test: vendor $(gosources)
	go test ./...

build: vendor $(gosources)
	go build ./...

clean:
	go clean
	rm -rf vendor

.PHONY: all build test clean
