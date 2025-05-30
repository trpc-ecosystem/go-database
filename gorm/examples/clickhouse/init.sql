CREATE DATABASE IF NOT EXISTS my_database;
USE my_database;

CREATE TABLE IF NOT EXISTS users (
    id UInt64,
    username String,
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;
