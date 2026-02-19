.PHONY: build test docs-check release-check release-dry-run release

BINARY := gflight

build:
	go build -o $(BINARY) ./cmd/gflight

test:
	go test ./...

docs-check:
	./scripts/docs-check.sh

release-check:
	./scripts/release-check.sh "$(VERSION)"

release-dry-run:
	./scripts/release.sh --dry-run "$(VERSION)"

release:
	./scripts/release.sh "$(VERSION)"
