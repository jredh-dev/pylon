VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := bin/pylon

.PHONY: build install clean test version

build:
	@mkdir -p bin
	go build $(LDFLAGS) -o $(BINARY) ./cmd/pylon/

install: build
	cp $(BINARY) $(shell go env GOPATH)/bin/pylon

test:
	go test ./...

clean:
	rm -f $(BINARY)

version:
	@echo $(VERSION)
