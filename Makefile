.PHONY: test unittest integrationtest staticcheck generate generate-sandbox sandbox-examples

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
		go run ./$$dir || exit 1; \
		echo ""; \
	done
