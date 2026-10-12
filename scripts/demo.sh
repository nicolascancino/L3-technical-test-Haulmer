#!/bin/bash
# Smoke test and demo for the Cards API: builds the binary, runs it against a
# throwaway Postgres from docker-compose.yml on the default port, exercises the main flows
# with curl (including an idempotency key replay), and tears everything down.
set -u

readonly port=8080
readonly db_port=5432
readonly base_url="http://localhost:${port}"
bin_dir=$(mktemp -d)
readonly bin_dir
readonly bin_path="${bin_dir}/cards-api"

api_pid=""
failures=0

cleanup() {
	if [ -n "$api_pid" ] && kill -0 "$api_pid" 2>/dev/null; then
		kill "$api_pid" 2>/dev/null
		wait "$api_pid" 2>/dev/null
	fi
	docker compose down -v >/dev/null 2>&1
	rm -rf "$bin_dir"
}
trap cleanup EXIT

echo "==> Stopping any container publishing port ${db_port}"
busy=$(docker ps -q --filter "publish=${db_port}")
if [ -n "$busy" ]; then
	docker stop $busy >/dev/null
fi

echo "==> Starting Postgres on port ${db_port}"
docker compose up -d --wait db >/dev/null

echo "==> Building the API"
go build -o "$bin_path" ./cmd/api

echo "==> Starting the API on port ${port}"
APP_ENV=local DB_HOST=localhost DB_PORT="$db_port" DB_NAME=cards DB_USER=postgres DB_PASSWORD=dev \
	DB_SSLMODE=disable PAN_SALT=demo-salt PORT="$port" LOG_LEVEL=warn "$bin_path" &
api_pid=$!

for _ in $(seq 1 30); do
	curl -sf "${base_url}/health" >/dev/null 2>&1 && break
	sleep 1
done

check() {
	local name="$1" want="$2" got="$3" body="$4"
	if [ "$want" = "$got" ]; then
		echo "PASS ${name} (status ${got})"
	else
		echo "FAIL ${name}: status = ${got}, want ${want}; body = ${body}"
		failures=$((failures + 1))
	fi
}

post() {
	local key="$1" body="$2"
	curl -s -o /tmp/demo-body -w '%{http_code}' -X POST "${base_url}/cards" \
		-H 'Content-Type: application/json' -H "Idempotency-Key: ${key}" -d "$body"
}

echo
echo "==> 1. GET /health"
status=$(curl -s -o /tmp/demo-body -w '%{http_code}' "${base_url}/health")
check "health check" 200 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 2. POST /cards, a valid Visa, key K1 (first time)"
status=$(post K1 '{"pan":"4111111111111111","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2030}')
check "create card" 201 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo
card_id=$(grep -o '"id":"[^"]*"' /tmp/demo-body | head -1 | cut -d'"' -f4)

echo
echo "==> 3. POST /cards again, same key K1, same request: idempotency replay"
status=$(post K1 '{"pan":"4111111111111111","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2030}')
check "replay returns 201" 201 "$status" "$(cat /tmp/demo-body)"
replay_id=$(grep -o '"id":"[^"]*"' /tmp/demo-body | head -1 | cut -d'"' -f4)
if [ "$replay_id" = "$card_id" ]; then
	echo "PASS replay returned the original card (id ${replay_id})"
else
	echo "FAIL replay returned a different card: ${replay_id}, want ${card_id}"
	failures=$((failures + 1))
fi
cat /tmp/demo-body
echo

echo
echo "==> 4. POST /cards, key K2, same PAN: rejected as a duplicate (unique pan_hash)"
status=$(post K2 '{"pan":"4111111111111111","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2030}')
check "duplicate pan" 409 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 5. POST /cards, without Idempotency-Key: rejected"
status=$(curl -s -o /tmp/demo-body -w '%{http_code}' -X POST "${base_url}/cards" \
	-H 'Content-Type: application/json' \
	-d '{"pan":"5555555555554444","cardHolderName":"Grace Hopper","expMonth":6,"expYear":2031}')
check "missing idempotency key" 400 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 6. POST /cards, PAN that fails Luhn"
status=$(post K3 '{"pan":"4111111111111112","cardHolderName":"Ada Lovelace","expMonth":12,"expYear":2030}')
check "invalid luhn" 422 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 7. POST /cards, expired card"
status=$(post K4 '{"pan":"5555555555554444","cardHolderName":"Grace Hopper","expMonth":1,"expYear":2020}')
check "expired card" 422 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 8. POST /cards, a second valid card, key K5"
status=$(post K5 '{"pan":"378282246310005","cardHolderName":"Grace Hopper","expMonth":3,"expYear":2033}')
check "create second card" 201 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 9. GET /cards/{id}"
status=$(curl -s -o /tmp/demo-body -w '%{http_code}' "${base_url}/cards/${card_id}")
check "get card" 200 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 10. GET /cards?limit=1 (cursor pagination)"
status=$(curl -s -o /tmp/demo-body -w '%{http_code}' "${base_url}/cards?limit=1")
check "list cards" 200 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
echo "==> 11. GET /cards:search?last4=1111"
status=$(curl -s -o /tmp/demo-body -w '%{http_code}' "${base_url}/cards:search?last4=1111")
check "search cards" 200 "$status" "$(cat /tmp/demo-body)"
cat /tmp/demo-body
echo

echo
if [ "$failures" -eq 0 ]; then
	echo "RESULT: PASS"
else
	echo "RESULT: FAIL (${failures} check(s) failed)"
fi
exit "$failures"
