.PHONY: build test test-race vet fmt tidy clean run

run:
	go run .

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	go clean ./...

# verify: full gate check required before any merge
verify: build test-race vet
	@echo "✓ All verification checks passed."
