#!/usr/bin/env bash
docker pull clickhouse:25.4.1
docker stop clickhouse-container || true
docker rm clickhouse-container || true

# https://hub.docker.com/_/clickhouse
docker run --name clickhouse-container \
  -e CLICKHOUSE_USER=my-username \
  -e CLICKHOUSE_PASSWORD=my-secret-pw\
  -d -p 9000:9000 \
  -v "$(pwd)"/init.sql:/docker-entrypoint-initdb.d/init.sql \
  clickhouse:25.4.1