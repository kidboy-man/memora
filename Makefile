BINARY=bin/memora
GO=go

.PHONY: build test test-integration test-e2e lint fmt fmt-check vet clean

build:
	$(GO) build -o $(BINARY) ./cmd/memora

test:
	$(GO) test -race ./...

test-integration:
	$(GO) test -race -tags integration ./...

test-e2e:
	$(GO) test -tags e2e -timeout 180s -v ./test/e2e/...

lint: vet fmt-check

vet:
	$(GO) vet ./...

fmt:
	$(GO) fmt ./...

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

clean:
	rm -rf bin/
