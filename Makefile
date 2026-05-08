.PHONY: ci vet test build zip clean

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
DIST    := dist

ci:
	cd ci && dagger run go run . --ci --version=$(VERSION) --output=../$(DIST)

vet:
	cd ci && dagger run go run . --vet --version=$(VERSION)

test:
	cd ci && dagger run go run . --test --version=$(VERSION)

build:
	cd ci && dagger run go run . --build --version=$(VERSION) --output=../$(DIST)

zip: build
	@cd $(DIST) && for f in nim-*; do \
		[ -f "$${f}.zip" ] && continue; \
		chmod +x "$$f"; \
		zip "$${f}.zip" "$$f"; \
		echo " ✓ $${f}.zip"; \
	done

clean:
	rm -rf $(DIST)
