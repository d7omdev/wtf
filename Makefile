BINARY := bin/wtf
GO := go

.PHONY: all build test vet fmt lint clean install doctor

all: build

build:
	$(GO) build -trimpath -ldflags "-s -w" -o $(BINARY) ./cmd/wtf

install: build
	install -m 0755 $(BINARY) $(HOME)/.local/bin/wtf

test:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

lint: vet
	@test -z "$$(gofmt -l .)" || { echo "gofmt needed"; gofmt -l .; exit 1; }

clean:
	rm -rf bin

doctor:
	./$(BINARY) doctor

release-check:
	./$(BINARY) version
	$(GO) test ./...
	$(GO) build ./...
