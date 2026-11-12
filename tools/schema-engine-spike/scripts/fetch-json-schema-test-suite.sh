#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
spike_dir="$(cd -- "$script_dir/.." && pwd)"
lock_file="$spike_dir/fixtures/JSON-Schema-Test-Suite.lock"
repository="$(sed -n 's/^repository=//p' "$lock_file")"
commit="$(sed -n 's/^commit=//p' "$lock_file")"
suite_root="${1:-$spike_dir/.cache/JSON-Schema-Test-Suite}"

if [[ ! "$commit" =~ ^[0-9a-f]{40}$ ]]; then
  echo "invalid pinned JSON-Schema-Test-Suite commit" >&2
  exit 2
fi

if [[ ! -d "$suite_root/.git" ]]; then
  git clone --filter=blob:none --no-checkout "$repository" "$suite_root"
fi
git -C "$suite_root" fetch --depth=1 origin "$commit"
git -C "$suite_root" checkout --detach "$commit"
actual="$(git -C "$suite_root" rev-parse HEAD)"
if [[ "$actual" != "$commit" ]]; then
  echo "suite checkout mismatch: expected $commit, got $actual" >&2
  exit 1
fi
printf '%s\n' "$suite_root"
