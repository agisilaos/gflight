.PHONY: build test docs-check smoke-real-provider release-check release-dry-run release

BINARY := gflight

build:
	go build -o $(BINARY) ./cmd/gflight

test:
	go test ./...

docs-check:
	./scripts/docs-check.sh

smoke-real-provider:
	./scripts/smoke-real-provider.sh

release-check:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release-check VERSION=v0.1.0)"; exit 2; fi
	./scripts/release-check.sh "$(VERSION)"

release-dry-run:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release-dry-run VERSION=v0.1.0)"; exit 2; fi
	./scripts/release.sh --dry-run "$(VERSION)"

release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release VERSION=v0.1.0)"; exit 2; fi
	./scripts/release.sh "$(VERSION)"
