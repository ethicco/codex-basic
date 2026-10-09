#!/usr/bin/env bash

set -euo pipefail

# Consume the event payload so the hook protocol can complete cleanly.
cat >/dev/null

project_root=$(git rev-parse --show-toplevel)
files=()

while IFS= read -r file; do
  case "$file" in
    frontend/*.js|frontend/*.jsx|frontend/*.ts|frontend/*.tsx|frontend/*.mjs|frontend/*.mts|frontend/*.cts)
      files+=("$project_root/$file")
      ;;
  esac
done < <(
  {
    git -C "$project_root" diff --name-only --diff-filter=ACMR -- frontend
    git -C "$project_root" ls-files --others --exclude-standard -- frontend
  } | sort -u
)

if [ "${#files[@]}" -eq 0 ]; then
  echo '{}'
  exit 0
fi

(
  cd "$project_root/frontend"
  npm exec eslint -- "${files[@]}"
)
echo '{}'
