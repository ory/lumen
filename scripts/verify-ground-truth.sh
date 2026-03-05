#!/usr/bin/env bash
# verify-ground-truth.sh — check that ground truth symbols exist in fixture files
#
# Usage: bash scripts/verify-ground-truth.sh
#
# For each ground truth file in testdata/ground-truth/, extracts symbol names
# from the "Key Types in Fixtures" section and verifies each appears in the
# corresponding fixture directory.
set -euo pipefail

REPO="$(cd "$(dirname "$0")/.." && pwd)"
GT_DIR="$REPO/testdata/ground-truth"
FX_DIR="$REPO/testdata/fixtures"

if [[ ! -d "$GT_DIR" ]]; then
  echo "No ground-truth directory found at $GT_DIR"
  exit 0
fi

# Map slug prefixes to fixture language directories
slug_to_lang() {
  case "$1" in
    go-*)     echo "go" ;;
    py-*)     echo "python" ;;
    ts-*)     echo "ts" ;;
    java-*)   echo "java" ;;
    js-*)     echo "js" ;;
    ruby-*)   echo "ruby" ;;
    rust-*)   echo "rust" ;;
    php-*)    echo "php" ;;
    *)        echo "" ;;
  esac
}

total_symbols=0
stale_symbols=0
checked_files=0

for gt_file in "$GT_DIR"/*.md; do
  [[ -f "$gt_file" ]] || continue
  slug=$(basename "$gt_file" .md)
  lang=$(slug_to_lang "$slug")

  if [[ -z "$lang" ]]; then
    echo "WARN: cannot determine language for $slug, skipping"
    continue
  fi

  lang_dir="$FX_DIR/$lang"
  if [[ ! -d "$lang_dir" ]]; then
    echo "WARN: fixture directory $lang_dir not found for $slug"
    continue
  fi

  checked_files=$((checked_files + 1))
  echo "=== $slug ($lang) ==="

  # Extract symbol names from "Key Types" section.
  # Looks for backtick-quoted names: `SymbolName` (filename)
  in_key_types=false
  file_stale=0
  file_total=0

  while IFS= read -r line; do
    # Detect section boundaries
    if [[ "$line" =~ ^##[[:space:]]+Key[[:space:]]+Types ]]; then
      in_key_types=true
      continue
    fi
    if [[ "$line" =~ ^##[[:space:]] ]] && $in_key_types; then
      break
    fi

    if $in_key_types; then
      # Extract backtick-quoted symbol names like `SymbolName`
      while [[ "$line" =~ \`([A-Za-z_][A-Za-z0-9_]*)\` ]]; do
        symbol="${BASH_REMATCH[1]}"
        # Remove matched symbol to find next one
        line="${line#*\`$symbol\`}"

        file_total=$((file_total + 1))
        total_symbols=$((total_symbols + 1))

        # Search for the symbol in fixture files
        if ! grep -rql "$symbol" "$lang_dir" >/dev/null 2>&1; then
          echo "  STALE: '$symbol' not found in $lang_dir/"
          stale_symbols=$((stale_symbols + 1))
          file_stale=$((file_stale + 1))
        fi
      done
    fi
  done < "$gt_file"

  if [[ $file_total -eq 0 ]]; then
    echo "  (no symbols extracted from Key Types section)"
  elif [[ $file_stale -eq 0 ]]; then
    echo "  OK: $file_total symbols verified"
  else
    echo "  $file_stale/$file_total symbols are stale"
  fi
done

echo ""
echo "========================================================"
echo "Checked: $checked_files ground truth files"
echo "Symbols: $total_symbols total, $stale_symbols stale"
echo "========================================================"

if [[ $stale_symbols -gt 0 ]]; then
  echo "FAIL: $stale_symbols stale symbols found"
  exit 1
fi
