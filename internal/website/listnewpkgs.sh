#!/usr/bin/env bash

# This script lists the package names that makeimports.sh would create
# _index.md files for, one per line.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
# except x is too verbose
set -euo pipefail

# Change into repository root.
cd "$(dirname "$0")/../.."
OUTDIR=internal/website/content

shopt -s nullglob # glob patterns that don't match turn into the empty string, instead of themselves

function files_exist() { # assumes nullglob
  [[ ${1:-""} != "" ]]
}

# Find all directories that do not begin with '.' or '_' or contain 'testdata'. Use the %P printf
# directive to remove the initial './'.
for pkg in $(find . -type d \( -name '[._]?*' -prune -o -name testdata -prune -o -printf '%P ' \)); do
  # Only consider directories that contain Go source files.
  outfile="$OUTDIR/$pkg/_index.md"
  if files_exist $pkg/*.go && [[ ! -e "$outfile" ]]; then
    echo "$pkg"
  fi
done
