#!/usr/bin/env bash

# Starts a local etcd instance via Docker.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
set -euo pipefail

# Clean up and run etcd.
echo "Starting etcd..."
docker rm -f etcd &> /dev/null || :
docker run -d -p 2379:2379 -p 4001:4001 --name etcd quay.io/coreos/etcd:v3.4.14 /usr/local/bin/etcd --advertise-client-urls http://0.0.0.0:2379 --listen-client-urls http://0.0.0.0:2379 &> /dev/null
echo "...done. Run \"docker rm -f etcd\" to clean up the container."
echo
