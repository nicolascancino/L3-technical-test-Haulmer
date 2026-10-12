.PHONY: build test vet openapi_http demo

ifeq ($(OS),Windows_NT)
BASH := "C:/Program Files/Git/bin/bash.exe"
else
BASH := bash
endif

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

openapi_http:
	@$(BASH) scripts/openapi-http.sh cards internal/ports ports

demo:
	@$(BASH) scripts/demo.sh
