.PHONY: build test vet openapi_http demo run down

ifeq ($(OS),Windows_NT)
BASH := "C:/Program Files/Git/bin/bash.exe"
else
BASH := bash
endif

-include .env
-include .env.local
export

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

run:
	docker compose up -d --wait db
	go run ./cmd/api

down:
	docker compose down -v
