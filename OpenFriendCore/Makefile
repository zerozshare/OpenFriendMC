BIN := openfriend
PKG := ./cmd/openfriend
DIST := dist
VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: build clean dist test fmt vet tidy

build:
	go build -ldflags '$(LDFLAGS)' -o $(BIN) $(PKG)

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN) $(DIST)

dist: clean
	@mkdir -p $(DIST)
	@$(foreach p,$(PLATFORMS), \
		os=$(word 1,$(subst /, ,$(p))); \
		arch=$(word 2,$(subst /, ,$(p))); \
		out=$(DIST)/$(BIN)-$$os-$$arch$$( [ "$$os" = "windows" ] && echo .exe ); \
		echo "Building $$out..."; \
		GOOS=$$os GOARCH=$$arch go build -ldflags '$(LDFLAGS)' -o $$out $(PKG) || exit 1; \
	)
	@ls -la $(DIST)
