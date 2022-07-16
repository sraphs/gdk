#!/usr/bin/env bash

# Starts a local redis instance via Docker.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
set -euo pipefail

# Clean up and run redis.
echo "Starting redis..."
docker rm -f redis &>/dev/null || :
docker run -d -p 6379:6379 --name redis redis:6.2.7 &>/dev/null
echo "...done. Run \"docker rm -f redis\" to clean up the container."
echo
