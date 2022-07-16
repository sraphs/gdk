#!/usr/bin/env bash

# Starts all local instances needed for Go CDK tests.
# You must have Docker installed.
# Run this script from the top level of the tree, e.g.:
#   ./internal/testing/start_local_deps.sh

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
set -euo pipefail

./pubsub/kafkapubsub/localkafka.sh
./pubsub/rabbitpubsub/localrabbit.sh
./runtimevar/etcdvar/localetcd.sh
./secrets/hashivault/localvault.sh

sleep 10
