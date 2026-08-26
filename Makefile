.PHONY: build test check-help docs-check smoke-real-provider changelog-context release-check release-check-ci release-dry-run release

BINARY := gflight

build:
	go build -o $(BINARY) ./cmd/gflight

test:
	go test ./...

check-help:
	./scripts/check-help.sh

docs-check:
	./scripts/docs-check.sh

smoke-real-provider:
	./scripts/smoke-real-provider.sh

changelog-context:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make changelog-context VERSION=v0.1.0)"; exit 2; fi
	./scripts/changelog-context.sh "$(VERSION)"

release-check:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release-check VERSION=v0.1.0)"; exit 2; fi
	./scripts/release-check.sh "$(VERSION)"

release-check-ci:
	./scripts/release-check.sh --ci

release-dry-run:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release-dry-run VERSION=v0.1.0)"; exit 2; fi
	./scripts/release.sh --dry-run "$(VERSION)"

release:
	@if [ -z "$(VERSION)" ]; then echo "VERSION is required (e.g. make release VERSION=v0.1.0)"; exit 2; fi
	./scripts/release.sh "$(VERSION)"
