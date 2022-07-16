#!/usr/bin/env bash

# Starts a local pulsar instance via Docker.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
set -euo pipefail

# Clean up and run pulsar.
echo "Starting pulsar..."
docker rm -f pulsar &>/dev/null || :
docker run -d -p 6650:6650 --name=pulsar apachepulsar/pulsar:2.10.1 bin/pulsar standalone &>/dev/null
echo "...done. Run \"docker rm -f pulsar\" to clean up the container."
echo
