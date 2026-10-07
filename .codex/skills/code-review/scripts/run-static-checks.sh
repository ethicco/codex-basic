#!/usr/bin/env sh
set -eu

if [ "$#" -gt 1 ]; then
  echo "Usage: $0 [<git-revision-or-range>]" >&2
  exit 2
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_dir/../../../.." && pwd)
revision=${1:-HEAD}
if [ "$#" -eq 0 ]; then
  changed_paths=$( {
    git -C "$repository_root" diff --name-only HEAD
    git -C "$repository_root" ls-files --others --exclude-standard
  } | sort -u)
else
  changed_paths=$(git -C "$repository_root" diff --name-only "$revision")
fi

if [ -z "$changed_paths" ]; then
  echo "No tracked changes in $revision; static checks skipped."
  exit 0
fi

if printf '%s\n' "$changed_paths" | grep -q '^backend/'; then
  echo '==> Backend: gofmt, vet, test, build'
  unformatted=$(find "$repository_root/backend" -name '*.go' -not -path "$repository_root/backend/docs/*" -exec gofmt -l {} +)
  if [ -n "$unformatted" ]; then
    printf '%s\n' 'gofmt is required for:' >&2
    printf '%s\n' "$unformatted" >&2
    exit 1
  fi
  (
    cd "$repository_root/backend"
    go vet ./...
    go test ./...
    go build ./...
  )
fi

if printf '%s\n' "$changed_paths" | grep -q '^frontend/'; then
  echo '==> Frontend: lint and production build'
  (
    cd "$repository_root/frontend"
    npm run lint
    npm run build
  )
fi

if ! printf '%s\n' "$changed_paths" | grep -qE '^(backend|frontend)/'; then
  echo 'No backend or frontend files changed; static checks skipped.'
fi
