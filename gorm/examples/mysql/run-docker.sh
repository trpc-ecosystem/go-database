#!/usr/bin/env bash
docker pull mysql:9.3.0
docker stop mysql-container || true
docker rm mysql-container || true

# https://hub.docker.com/_/mysql/
docker run --name mysql-container \
  -e MYSQL_ROOT_PASSWORD=my-secret-pw \
  -d -p 3306:3306 \
  -v "$(pwd)"/init.sql:/docker-entrypoint-initdb.d/init.sql \
  mysql:9.3.0