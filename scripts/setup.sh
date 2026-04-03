#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
ln -sf "$SCRIPT_DIR/pre-push" "$REPO_ROOT/.git/hooks/pre-push"
echo "Git hooks installed."
