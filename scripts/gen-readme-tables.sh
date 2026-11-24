#!/usr/bin/env bash
# Regenerates the "Supported Resources" / "Supported Data Sources" tables in
# README.md from the docs/ pages tfplugindocs already generates, so the README
# can't drift from the actual set of resources/data sources. Run after `make docs`
# (the Makefile's `docs` target calls this automatically).
set -euo pipefail
cd "$(dirname "$0")/.."

# gen_rows DIR KIND prints one markdown table row per *.md file in DIR, where
# KIND is "Resource" or "Data Source" (must match the page_title suffix tfplugindocs writes).
gen_rows() {
  local dir=$1 kind=$2
  local f name desc
  for f in "$dir"/*.md; do
    name=$(sed -nE 's/^page_title: "([a-zA-Z0-9_]+) '"${kind}"' .*/\1/p' "$f")
    desc=$(awk '/^description: \|-$/{getline; sub(/^  /, ""); print; exit}' "$f")
    [ -n "$name" ] || continue
    printf '| `%s` | %s |\n' "$name" "$desc"
  done | LC_ALL=C sort
}

# replace_block MARKER CONTENT replaces everything between
# "<!-- MARKER:start -->" and "<!-- MARKER:end -->" in README.md with CONTENT
# (the header row(s) plus generated rows), keeping the marker lines themselves.
replace_block() {
  local marker=$1 content=$2
  local start end tmp
  start=$(grep -n "<!-- ${marker}:start -->" README.md | head -1 | cut -d: -f1)
  end=$(grep -n "<!-- ${marker}:end -->" README.md | head -1 | cut -d: -f1)
  if [ -z "$start" ] || [ -z "$end" ]; then
    echo "gen-readme-tables: markers for '${marker}' not found in README.md" >&2
    exit 1
  fi
  tmp=$(mktemp)
  { head -n "$start" README.md; printf '%s\n' "$content"; tail -n "+$end" README.md; } > "$tmp"
  mv "$tmp" README.md
}

resources_table=$(printf '| Resource | Purpose |\n|---|---|\n%s' "$(gen_rows docs/resources Resource)")
datasources_table=$(printf '| Data Source | Purpose |\n|---|---|\n%s' "$(gen_rows docs/data-sources "Data Source")")

replace_block resources "$resources_table"
replace_block datasources "$datasources_table"

echo "README.md tables regenerated from docs/."
