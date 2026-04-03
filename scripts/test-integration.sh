#!/usr/bin/env bash
set -euo pipefail

# Load .env if present (export vars, skip comments and blank lines)
if [ -f .env ]; then
  while IFS= read -r line || [ -n "$line" ]; do
    # Skip comments and blank lines
    [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
    export "$line"
  done < .env
fi

if [ -z "${NASA_API_KEY:-}" ]; then
  echo "ERROR: NASA_API_KEY is not set." >&2
  echo "Create a .env file with NASA_API_KEY=your-key or export it directly." >&2
  echo "Get a free key at https://api.nasa.gov/" >&2
  exit 1
fi

echo "Running integration tests..."
go test -tags integration -count=1 -timeout 10m -v ./...
