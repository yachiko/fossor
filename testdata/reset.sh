#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Removing existing test repos..."
rm -rf "$SCRIPT_DIR/repos" "$SCRIPT_DIR/remotes"

# Ensure git trusts repos in this directory
git config --global --add safe.directory '*' 2>/dev/null || true

echo "Recreating..."
exec "$SCRIPT_DIR/setup.sh"
