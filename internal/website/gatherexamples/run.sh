#!/usr/bin/env bash

# Run gatherexamples for the project.
# The output of this script should be piped into
# internal/website/data/examples.json, where it's picked up by Hugo.

set -eo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/../../.."
sed -e '/^#/d' -e '/^$/d' allmodules | awk '{print $1}' | xargs go run internal/website/gatherexamples/gatherexamples.go
