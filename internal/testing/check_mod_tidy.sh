#!/usr/bin/env bash

# This script checks to see if `go mod tidy` has been run on the module
# in the current directory.
#
# It exits with status 1 if "go mod tidy && go list -deps ./..." would
# make changes.
#
# TODO(rvangent): Replace this with `go mod tidy --check` when it exists:
# https://github.com/golang/go/issues/27005.
#
# TODO(rvangent): Drop the "go list" part here and in gomodcleanup.sh once
# https://github.com/golang/go/issues/31248 is fixed.

set -euo pipefail

TMP_GOMOD=$(mktemp)
TMP_GOSUM=$(mktemp)

function cleanup() {
  # Restore the original files in case "go mod tidy" made changes.
  if [[ -f "$TMP_GOMOD" ]]; then
    mv "$TMP_GOMOD" ./go.mod
  fi
  if [[ -f "$TMP_GOSUM" ]]; then
    mv "$TMP_GOSUM" ./go.sum
  fi
}
trap cleanup EXIT

# Make copies of the current files.
cp ./go.mod "$TMP_GOMOD"
cp ./go.sum "$TMP_GOSUM"

# Modifies the files in-place.
go mod tidy
go list -deps ./... &>/dev/null

# Check for diffs.
diff -u "$TMP_GOMOD" ./go.mod
diff -u "$TMP_GOSUM" ./go.sum
