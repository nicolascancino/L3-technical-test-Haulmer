.PHONY: build test vet openapi_http demo run down

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

run: export APP_ENV = local
run: export DB_HOST = localhost
run: export DB_PORT = 5432
run: export DB_NAME = cards
run: export DB_USER = postgres
run: export DB_PASSWORD = dev
run: export DB_SSLMODE = disable
run: export PAN_SALT = local-salt
run:
	docker compose up -d --wait db
	go run ./cmd/api

down:
	docker compose down -v
