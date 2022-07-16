#!/usr/bin/env bash

# This script generates Markdown files that will include <meta> suitable for
# "go get"'s import path redirection feature (see
# https://golang.org/cmd/go/#hdr-Remote_import_paths) in the final Hugo output.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
# except x is too verbose
set -euo pipefail

# Change into repository root.
cd "$(dirname "$0")/../.."
OUTDIR=internal/website/content

for pkg in $(internal/website/listnewpkgs.sh); do
  # Only consider directories that contain Go source files.
  outfile="$OUTDIR/$pkg/_index.md"
  mkdir -p "$OUTDIR/$pkg"
  echo "Generating github.com/sraphs/gdk/$pkg"
  echo "---" >>"$outfile"
  echo "title: github.com/sraphs/gdk/$pkg" >>"$outfile"
  echo "type: pkg" >>"$outfile"
  echo "---" >>"$outfile"
done
