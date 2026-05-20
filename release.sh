#!/usr/bin/env bash
# release.sh — raycanvas release script
#
# Usage:
#   ./release.sh <version>          Full release zip
#   ./release.sh <version> --short  Quick checkpoint zip (skips git tag prompt)
#
# What it does:
#   1. Validates the version argument
#   2. Checks CHANGELOG.md top entry matches the version
#   3. Updates VERSION and pkg/version/version.go
#   4. Runs go vet on library and all examples
#   5. Packages a zip named raycanvas-<version>.zip
#   6. Prints a summary and the zip path
#
# Version format:
#   Semver:  0.1.0  0.2.0  1.0.0
#   Patched: 0.1.0-patched01  (incremental checkpoint, no CHANGELOG entry needed)
#
# Release hygiene rules (from project working agreement):
#   - CHANGELOG top entry must match version (except --short / patched builds)
#   - VERSION, pkg/version/version.go, and zip filename must all agree
#   - Never manually zip — always use this script
#   - After running, present the zip to the user via present_files

set -euo pipefail

# ── Arguments ────────────────────────────────────────────────────────────────
VERSION_ARG="${1:-}"
SHORT="${2:-}"

if [[ -z "$VERSION_ARG" ]]; then
  echo "Usage: $0 <version> [--short]" >&2
  exit 1
fi

# Validate format: semver or semver-patchedNN
if ! echo "$VERSION_ARG" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-patched[0-9]+)?$'; then
  echo "Error: version must match semver (e.g. 0.1.0) or semver-patchedNN (e.g. 0.1.0-patched01)" >&2
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$REPO_ROOT"

IS_PATCHED=false
if echo "$VERSION_ARG" | grep -q '\-patched'; then
  IS_PATCHED=true
fi

echo "==> raycanvas release: $VERSION_ARG"

# ── Step 1: CHANGELOG check ──────────────────────────────────────────────────
# For full (non-patched) releases, the top ## entry must match the version.
# For patched builds or --short, we skip this check.
if [[ "$IS_PATCHED" == "false" && "$SHORT" != "--short" ]]; then
  TOP_ENTRY=$(grep -m1 '^## ' CHANGELOG.md | sed 's/^## //' | awk '{print $1}')
  if [[ "$TOP_ENTRY" != "$VERSION_ARG" ]]; then
    echo "Error: CHANGELOG.md top entry is '$TOP_ENTRY', expected '$VERSION_ARG'" >&2
    echo "Update CHANGELOG.md before releasing." >&2
    exit 1
  fi
  echo "    CHANGELOG.md top entry: $TOP_ENTRY ✓"
else
  echo "    CHANGELOG check: skipped (patched or --short)"
fi

# ── Step 2: Update VERSION ────────────────────────────────────────────────────
echo "$VERSION_ARG" > VERSION
echo "    VERSION: $VERSION_ARG ✓"

# ── Step 3: Update pkg/version/version.go ────────────────────────────────────
VERSION_FILE="pkg/version/version.go"
cat > "$VERSION_FILE" << GOEOF
// Package version exposes the raycanvas library version.
// This file is kept in sync with the VERSION file at the repository root
// by release.sh. Do not edit manually — use release.sh to update.
package version

// Version is the current raycanvas release version.
// Format: semver for milestone releases (0.1.0, 0.2.0, 1.0.0).
// Incremental patch builds append a suffix: 0.1.0-patched01, etc.
const Version = "$VERSION_ARG"
GOEOF
echo "    pkg/version/version.go: $VERSION_ARG ✓"

# ── Step 4: Consistency check ─────────────────────────────────────────────────
# Read back and verify all three sources agree.
FILE_VERSION="$(cat VERSION | tr -d '[:space:]')"
GO_VERSION="$(grep 'const Version' pkg/version/version.go | grep -oP '"[^"]+"' | tr -d '"')"

if [[ "$FILE_VERSION" != "$VERSION_ARG" ]]; then
  echo "Error: VERSION file mismatch: got '$FILE_VERSION'" >&2
  exit 1
fi
if [[ "$GO_VERSION" != "$VERSION_ARG" ]]; then
  echo "Error: version.go mismatch: got '$GO_VERSION'" >&2
  exit 1
fi
echo "    Consistency check: VERSION == version.go == $VERSION_ARG ✓"

# ── Step 5: go vet ───────────────────────────────────────────────────────────
echo "==> Running go vet..."
go vet ./... 2>&1
for ex in examples/*/; do
  ex_name="$(basename "$ex")"
  [[ "$ex_name" == "internal" ]] && continue
  (cd "$ex" && go vet ./...) 2>&1 && echo "    vet $ex_name ✓"
done
echo "    go vet: all clean ✓"

# ── Step 6: Package zip ───────────────────────────────────────────────────────
ZIP_NAME="raycanvas-${VERSION_ARG}.zip"
ZIP_DIR="raycanvas-${VERSION_ARG}"

echo "==> Packaging $ZIP_NAME..."

# Build file list: everything tracked, excluding outputs and temp files.
# We include:
#   *.go  go.mod  go.sum  VERSION  CHANGELOG.md  ARCHITECTURE.md  Makefile
#   release.sh  package.sh
#   examples/**/main.go  examples/**/go.mod  examples/**/go.sum
#   pkg/version/version.go

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

STAGE="$TMP_DIR/$ZIP_DIR"
mkdir -p "$STAGE"

# Library root
cp VERSION CHANGELOG.md ARCHITECTURE.md README.md Makefile release.sh package.sh \
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

# Example binaries
for ex in examples/*/; do
  ex_name="$(basename "$ex")"
  [[ "$ex_name" == "internal" ]] && continue
  mkdir -p "$STAGE/examples/$ex_name"
  cp "examples/$ex_name/main.go" "$STAGE/examples/$ex_name/"
  cp "examples/$ex_name/go.mod"  "$STAGE/examples/$ex_name/"
  [[ -f "examples/$ex_name/go.sum" ]] && \
    cp "examples/$ex_name/go.sum" "$STAGE/examples/$ex_name/" || true
done

# Zip
(cd "$TMP_DIR" && zip -r "$REPO_ROOT/$ZIP_NAME" "$ZIP_DIR") > /dev/null

# Verify zip
ZIP_SIZE="$(du -h "$ZIP_NAME" | cut -f1)"
echo "    $ZIP_NAME ($ZIP_SIZE) ✓"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "raycanvas $VERSION_ARG released."
echo "Zip: $REPO_ROOT/$ZIP_NAME"
echo ""
echo "Next steps:"
if [[ "$IS_PATCHED" == "false" && "$SHORT" != "--short" ]]; then
  echo "  git add VERSION CHANGELOG.md pkg/version/version.go"
  echo "  git commit -m \"release: $VERSION_ARG\""
  echo "  git tag v$VERSION_ARG"
  echo "  git push && git push --tags"
fi
echo "  present_files $ZIP_NAME"
