#!/usr/bin/env bash
rm mysqlite.db || true
docker pull alpine/sqlite:3.48.0
docker stop sqlite-container || true
docker rm sqlite-container || true

# https://hub.docker.com/r/alpine/sqlite
docker run --name sqlite-container \
  -v "$(pwd)":/apps \
  -w /apps alpine/sqlite:3.48.0 mysqlite.db "CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT);" \
