#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../.."

GOCACHE_DIR="${GOCACHE:-}"
if [[ -z "${GOCACHE_DIR}" ]]; then
  GOCACHE_DIR="$(mktemp -d)"
  trap 'rm -rf "${GOCACHE_DIR}"' EXIT
  export GOCACHE="${GOCACHE_DIR}"
fi

go run ./internal/tools/parity_report \
  -out reports/parity_report.md \
  -json reports/parity_report.json

go test ./internal/notify -run TestAttachmentRequestParity -count=1 -v

# Exercise every HTTP schema option as well as the curated provider cases.
go test ./internal/parity -tags=e2e -run '^TestE2E' -count=1 \
  -timeout="${APPRISE_PARITY_TIMEOUT:-30m}"
