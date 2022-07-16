#!/usr/bin/env bash

# This script should be run from the root directory.
# It runs "go get -u && go mod tidy" on all modules in
# the repo, to update dependencies. Run runchecks.sh afterwards.
set -euo pipefail

sed -e '/^#/d' -e '/^$/d' allmodules | awk '{print $1}' | while read -r path || [[ -n "$path" ]]; do
  echo "updating $path"
  (cd "$path" && go get -u ./... &>/dev/null && go mod tidy &>/dev/null || echo "  FAILED! (some modules without code, like samples, are expected to fail)")
done
