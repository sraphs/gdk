#!/usr/bin/env bash

# Starts a local Kafka instance (plus supporting Zookeeper) via Docker.

# https://coderwall.com/p/fkfaqq/safer-bash-scripts-with-set-euxo-pipefail
set -euo pipefail

# Clean up and run Zookeeper.
echo "Starting Zookeeper (for Kafka)..."
docker rm -f zookeeper &>/dev/null || :
docker run -d --name=zookeeper -e ZOOKEEPER_CLIENT_PORT=2181 confluentinc/cp-zookeeper:6.0.1 &>/dev/null
echo "...done. Run \"docker rm -f zookeeper\" to clean up the container."
echo

# Clean up and run Kafka.
echo "Starting Kafka..."
docker rm -f kafka &>/dev/null || :
docker run -d -p 9092:9092 --name=kafka --link zookeeper -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 -e KAFKA_AUTO_CREATE_TOPICS_ENABLE=false -e KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=100 confluentinc/cp-kafka:6.0.1 &>/dev/null
echo "...done. Run \"docker rm -f kafka\" to clean up the container."
echo
