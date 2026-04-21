.PHONY: test unittest integrationtest staticcheck generate generate-sandbox sandbox-examples build install clean xgo-gen

# --- CLI ---------------------------------------------------------------------

# Output binary for `make build`. Override with `make build OUT=/usr/local/bin/sufy`.
OUT ?= ./sufy

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
