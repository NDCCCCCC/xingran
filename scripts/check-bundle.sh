#!/usr/bin/env bash
# scripts/check-bundle.sh
# Asserts that the build produced the required vendor chunks.
# Used to verify D-07 chunk strategy is working after Vite config changes.

set -euo pipefail

DIST="xingran-react-frontend/dist/assets"
REQUIRED=("vendor-react" "vendor-antd" "vendor-echarts" "vendor-three")
MISSING=()

for chunk in "${REQUIRED[@]}"; do
  if ! ls "$DIST"/${chunk}-*.js 2>/dev/null | head -1 | grep -q .; then
    MISSING+=("$chunk")
  fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
  echo "ERROR: Missing required vendor chunks: ${MISSING[*]}"
  echo "Expected chunks: ${REQUIRED[*]}"
  echo "Actual files in $DIST:"
  ls "$DIST" 2>/dev/null | grep vendor || echo "(no vendor-* chunks found)"
  exit 1
fi

echo "OK: All required vendor chunks present:"
for chunk in "${REQUIRED[@]}"; do
  size=$(ls -la "$DIST"/${chunk}-*.js 2>/dev/null | head -1 | awk '{print $5}')
  printf "  %-18s %s bytes\n" "$chunk:" "${size}"
done