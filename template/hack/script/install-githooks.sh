#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
cd "$ROOT_DIR"

chmod +x .githooks/pre-commit
git config core.hooksPath .githooks

echo "Installed git hooks:"
echo "  core.hooksPath=.githooks"
echo "Pre-commit gate is now active."
