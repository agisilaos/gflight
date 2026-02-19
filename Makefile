.PHONY: build test release-check release-dry-run release

BINARY := gflight

build:
	go build -o $(BINARY) ./cmd/gflight

test:
	go test ./...

release-check:
	./scripts/release-check.sh "$(VERSION)"

release-dry-run:
	./scripts/release.sh --dry-run "$(VERSION)"

release:
	./scripts/release.sh "$(VERSION)"
