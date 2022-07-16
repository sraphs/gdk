#!/usr/bin/env bash

set -euo pipefail

# To run this script manually to update alldeps:
#
# $ internal/testing/listdeps.sh > internal/testing/alldeps
#
# Make sure to use the same version of Go as used by tests
# (see .github/actions/tests.yml) when updating the alldeps file.
tmpfile=$(mktemp)
function cleanup() {
  rm -rf "$tmpfile"
}
trap cleanup EXIT

sed -e '/^#/d' -e '/^$/d' allmodules | awk '{print $1}' | while read -r path || [[ -n "$path" ]]; do
  (cd "$path" && go list -mod=readonly -deps -f '{{with .Module}}{{.Path}}{{end}}' ./... >>"$tmpfile")
done

# Sort using the native byte values to keep results from different environment consistent.
LC_ALL=C sort "$tmpfile" | uniq
