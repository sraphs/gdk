#!/usr/bin/env bash

# This script should be run from the root directory.
# It runs "go mod tidy && go list -deps ./..." on all modules in
# the repo, to ensure that go.mod and go.sum are in the canonical
# form that tests will verify (see check_mod_tidy.sh).
set -euo pipefail

sed -e '/^#/d' -e '/^$/d' allmodules | awk '{print $1}' | while read -r path || [[ -n "$path" ]]; do
  echo "cleaning up $path"
  (cd "$path" && go mod tidy && go list -deps ./... &>/dev/null || echo "  FAILED!")
done
