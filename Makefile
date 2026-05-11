.PHONY: test unittest integrationtest integrationtest-env staticcheck generate generate-sandbox sandbox-examples build install clean xgo-gen

# .env file used by *-env targets to inject test credentials.
ENV_FILE ?= .env

# --- CLI ---------------------------------------------------------------------

# Output binary for `make build`. Override with `make build OUT=/usr/local/bin/sufy`.
OUT ?= ./bin/sufy

# Regenerate xgo_autogen.go from cmd/sufy/*.gox. Required after editing any .gox file.
xgo-gen:
	xgo build ./cmd/sufy/

# Build the sufy CLI binary. Runs xgo codegen first so .gox edits are picked up.
build: xgo-gen
	go build -o $(OUT) ./cmd/sufy/

# Install the sufy CLI into $GOPATH/bin (or $GOBIN).
install: xgo-gen
	go install ./cmd/sufy/

# Remove the local ./sufy binary produced by `make build`.
clean:
	rm -f ./sufy

# --- Tests / codegen / examples ---------------------------------------------


test:
	go test -tags=unit -failfast -count=1 -v -timeout 30m -coverprofile=coverage.txt \
		$$(go list ./... | grep -v 'examples')

unittest:
	go test -tags=unit -failfast -count=1 -v -coverprofile=coverage.txt \
		$$(go list ./... | grep -v 'examples')

integrationtest:
	go test -tags=integration -failfast -count=1 -parallel 1 -v -coverprofile=coverage.txt \
		$$(go list ./... | grep -v 'examples')

# Run integration tests with environment variables loaded from $(ENV_FILE).
# Usage: make integrationtest-env  (uses .env by default)
#        make integrationtest-env ENV_FILE=.env.local
integrationtest-env:
	@test -f $(ENV_FILE) || { echo "missing env file: $(ENV_FILE)"; exit 1; }
	set -a; . ./$(ENV_FILE); set +a; \
		go test -tags=integration -failfast -count=1 -parallel 1 -v -coverprofile=coverage.txt \
			$$(go list ./... | grep -v 'examples')

staticcheck:
	staticcheck $$(go list ./... | grep -v 'examples')

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
