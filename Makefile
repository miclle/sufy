.PHONY: test unittest integrationtest staticcheck generate generate-sandbox sandbox-examples build install clean xgo-gen

# --- CLI ---------------------------------------------------------------------

# Output binary for `make build`. Override with `make build OUT=/usr/local/bin/sufy`.
OUT ?= ./sufy
# Output binary for the spf13/cobra implementation.
OUT_SPF13 ?= ./sufy-spf13

# Regenerate xgo_autogen.go from cmd/sufy/*.gox. Required after editing any .gox file.
xgo-gen:
	xgo build ./cmd/sufy/

# Build the sufy CLI binary. Runs xgo codegen first so .gox edits are picked up.
build: xgo-gen
	go build -o $(OUT) ./cmd/sufy/

# Build the parallel spf13/cobra implementation for comparison.
build-spf13:
	go build -o $(OUT_SPF13) ./cmd/sufy-spf13/

# Build both CLI variants.
build-all: build build-spf13

# Install the sufy CLI into $GOPATH/bin (or $GOBIN).
install: xgo-gen
	go install ./cmd/sufy/

# Remove the local ./sufy and ./sufy-spf13 binaries produced by build targets.
clean:
	rm -f ./sufy ./sufy-spf13

# --- Tests / codegen / examples ---------------------------------------------


test:
	go test -tags=unit -failfast -count=1 -v -timeout 30m -coverprofile=coverage.txt \
		$$(go list ./... | grep -v 'examples\|cmd')

unittest:
	go test -tags=unit -failfast -count=1 -v -coverprofile=coverage.txt \
		$$(go list ./... | grep -v 'examples\|cmd')

integrationtest:
	go test -tags=integration -failfast -count=1 -parallel 1 -v -coverprofile=coverage.txt \
		$$(go list ./... | grep -v 'examples\|cmd')

staticcheck:
	staticcheck $$(go list ./... | grep -v 'examples\|cmd')

generate: generate-sandbox

generate-sandbox:
	# Control plane API
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
		--config sandbox/internal/apis/oapi-codegen.yaml \
		api/sandbox/openapi.yml
	# envd HTTP API
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.4.1 \
		--config sandbox/internal/envdapi/oapi-codegen.yaml \
		api/sandbox/envd/envd.yaml
	# envd ConnectRPC (requires buf, protoc-gen-go, protoc-gen-connect-go)
	cd api/sandbox/envd && buf generate
	# Verify build
	go build ./sandbox/...

sandbox-examples:
	@for dir in examples/sandbox_*/; do \
		name=$$(basename $$dir); \
		echo "=== $$name ==="; \
		go run ./$$dir $(if $(AUTH),-auth $(AUTH)) || exit 1; \
		echo ""; \
	done
