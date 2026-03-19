#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")" && pwd)"

for mod in "$root"/internal/integrations/*/go.mod; do
  dir="$(dirname "$mod")"
  echo "==> $(basename "$dir")"
  go -C "${dir}" get -v -u ./...
  go -C "${dir}" mod tidy

done
