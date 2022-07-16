#!/usr/bin/env bash

# Starts a local RabbitMQ instance via Docker.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
set -euo pipefail

echo "Starting RabbitMQ..."
docker rm -f rabbit &> /dev/null || :
docker run -d --name rabbit -p 5672:5672 rabbitmq:3.8.9 &> /dev/null
echo "...done. Run \"docker rm -f rabbit\" to clean up the container."
echo
