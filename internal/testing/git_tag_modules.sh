#!/usr/bin/env bash

# This script should be run from the root directory.
# It creates git tags for all marked modules listed in the allmodules file.
set -euo pipefail

function usage() {
  echo
  echo "Usage: git_tag_modules.sh vX.X.X" 1>&2
  echo "  vX.X.X: the git tag version"
  exit 64
}

if [[ $# -ne 1 ]]; then
  echo "Need at least one argument"
  usage
fi
version="$1"

sed -e '/^#/d' -e '/^$/d' allmodules | awk '{ print $1, $2}' | while read -r path update || [[ -n "$path" ]]; do
  if [[ "$update" != "yes" ]]; then
    echo "$path is not marked to be released"
    continue
  fi

  tag="$version"
  if [[ "$path" != "." ]]; then
    tag="$path/$version"
  fi
  echo "Creating tag: ${tag}"
  git tag "$tag"
done
