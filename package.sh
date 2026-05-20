#!/usr/bin/env bash
# package.sh — raycanvas quick-package script
#
# Packages the current state of the repository into a zip without
# bumping the version. Use this to share work-in-progress or create
# a checkpoint before starting a risky change.
#
# For a proper versioned release, use release.sh instead.
#
# Usage:
#   ./package.sh              Uses current VERSION as the zip name
#   ./package.sh <suffix>     Appends suffix: raycanvas-0.1.0-<suffix>.zip

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO_ROOT"

CURRENT_VERSION="$(cat VERSION | tr -d '[:space:]')"
SUFFIX="${1:-}"

if [[ -n "$SUFFIX" ]]; then
  ZIP_LABEL="${CURRENT_VERSION}-${SUFFIX}"
else
  ZIP_LABEL="$CURRENT_VERSION"
fi

ZIP_NAME="raycanvas-${ZIP_LABEL}.zip"
ZIP_DIR="raycanvas-${ZIP_LABEL}"

echo "==> raycanvas package: $ZIP_NAME (version on disk: $CURRENT_VERSION)"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

STAGE="$TMP_DIR/$ZIP_DIR"
mkdir -p "$STAGE"

# Library root
cp VERSION CHANGELOG.md ARCHITECTURE.md Makefile release.sh package.sh \
   go.mod go.sum \
   "$STAGE/"

# Library source
cp *.go "$STAGE/"

# pkg/version
mkdir -p "$STAGE/pkg/version"
cp pkg/version/version.go "$STAGE/pkg/version/"

# Examples
# Internal packages
mkdir -p "$STAGE/examples/internal/fonts"
cp examples/internal/fonts/fonts.go "$STAGE/examples/internal/fonts/"
cp examples/internal/fonts/go.mod   "$STAGE/examples/internal/fonts/"
[[ -f examples/internal/fonts/go.sum ]] && \
  cp examples/internal/fonts/go.sum "$STAGE/examples/internal/fonts/" || true
cp examples/internal/fonts/*.ttf    "$STAGE/examples/internal/fonts/"

for ex in examples/*/; do
  ex_name="$(basename "$ex")"
  [[ "$ex_name" == "internal" ]] && continue
  mkdir -p "$STAGE/examples/$ex_name"
  cp "examples/$ex_name/main.go" "$STAGE/examples/$ex_name/"
  cp "examples/$ex_name/go.mod"  "$STAGE/examples/$ex_name/"
  [[ -f "examples/$ex_name/go.sum" ]] && \
    cp "examples/$ex_name/go.sum" "$STAGE/examples/$ex_name/" || true
done

(cd "$TMP_DIR" && zip -r "$REPO_ROOT/$ZIP_NAME" "$ZIP_DIR") > /dev/null

ZIP_SIZE="$(du -h "$ZIP_NAME" | cut -f1)"
echo "    $ZIP_NAME ($ZIP_SIZE) ✓"
echo "Zip: $REPO_ROOT/$ZIP_NAME"
