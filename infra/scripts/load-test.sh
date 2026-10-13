#!/usr/bin/env bash
# Ephemeral load test: apply the stacks, hit GET /health through the ALB,
# save the evidence and always destroy everything on exit.
#
# Usage: ENV=dev infra/scripts/load-test.sh
# Optional: REQUESTS (2000), CONCURRENCY (50), HEALTH_TIMEOUT seconds (600),
#           EVIDENCE_DIR (evidence/aws), BASE_URL (skips the terraform output lookup).
set -euo pipefail

ENV="${ENV:-dev}"
REQUESTS="${REQUESTS:-2000}"
CONCURRENCY="${CONCURRENCY:-50}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-600}"
EVIDENCE_DIR="${EVIDENCE_DIR:-evidence/aws}"

# Creation order; destruction runs in reverse.
STACKS=(network secrets rds alb ecs cloudwatch)

if [ "$ENV" = "prod" ]; then
  echo "Refusing to run an ephemeral load test against prod" >&2
  exit 1
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tf() { terraform -chdir="$root/infra/$1/$ENV" "${@:2}"; }

destroy() {
  local status=0 i
  echo "==> Destroying stacks (reverse order)"
  for ((i = ${#STACKS[@]} - 1; i >= 0; i--)); do
    tf "${STACKS[$i]}" init -input=false >/dev/null || status=1
    tf "${STACKS[$i]}" destroy -input=false -auto-approve || status=1
  done
  exit "$status"
}
# Runs on success, failure and Ctrl+C, so nothing is left billing.
trap destroy EXIT

echo "==> Applying stacks: ${STACKS[*]}"
for stack in "${STACKS[@]}"; do
  tf "$stack" init -input=false
  tf "$stack" apply -input=false -auto-approve
done

# The alb stack must expose an output named dns_name.
base_url="${BASE_URL:-$(tf alb output -raw dns_name)}"
case "$base_url" in http*) ;; *) base_url="http://$base_url" ;; esac

echo "==> Waiting for $base_url/health (up to ${HEALTH_TIMEOUT}s)"
deadline=$((SECONDS + HEALTH_TIMEOUT))
until curl -skf "$base_url/health" >/dev/null; do
  if ((SECONDS >= deadline)); then
    echo "Targets never became healthy" >&2
    exit 1
  fi
  sleep 5
done

mkdir -p "$root/$EVIDENCE_DIR"
out="$root/$EVIDENCE_DIR"

echo "==> hey: $REQUESTS requests, $CONCURRENCY concurrent"
docker run --rm williamyeh/hey -disable-compression -n "$REQUESTS" -c "$CONCURRENCY" "$base_url/health" \
  >"$out/hey-load.txt" 2>&1

echo "==> Distribution per instance"
seq "$REQUESTS" | xargs -P "$CONCURRENCY" -I{} curl -sk "$base_url/health" -w '\n' |
  sed '/^$/d' | sort | uniq -c >"$out/distribution-under-load.txt"

cat "$out/distribution-under-load.txt"
