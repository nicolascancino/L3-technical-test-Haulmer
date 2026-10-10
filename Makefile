.PHONY: build test vet openapi_http

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

openapi_http:
	@./scripts/openapi-http.sh cards internal/ports ports
