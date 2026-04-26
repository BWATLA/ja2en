.PHONY: build install test clean fmt vet lint check

BINARY := ja2en
VERSION := 0.5.0
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: fmt vet lint test

clean:
	rm -f $(BINARY)
	go clean
