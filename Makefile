BINARY   := bin/llmclientwrapper
CMD      := ./src/cmd
MODEL    ?= haiku-4.5
SYSTEM_FILE := ./system_prompt.md
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-X llmclientwrapper/src/internal/version.Version=$(VERSION)"

.PHONY: all build run test cover cover-summary vet clean

all: vet test build

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

run:
	go run $(CMD) --model $(MODEL) --system-file $(SYSTEM_FILE)

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

cover-summary:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

clean:
	rm -rf bin coverage.out coverage.html
