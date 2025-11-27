-- Drop and recreate the database
DROP DATABASE IF EXISTS keepalive;
CREATE DATABASE keepalive;

-- Create user if not exists
DO
$do$
BEGIN
   IF NOT EXISTS (
       SELECT FROM pg_catalog.pg_roles WHERE rolname = 'keepalive'
   ) THEN
CREATE ROLE keepalive LOGIN PASSWORD 'keepalive';
END IF;
END
$do$;

-- Grant privileges on the database
GRANT ALL PRIVILEGES ON DATABASE keepalive TO keepalive;

-- Connect to the database
\c keepalive

-- Ensure the user can use the public schema
GRANT USAGE ON SCHEMA public TO keepalive;
GRANT CREATE ON SCHEMA public TO keepalive;

-- Create the table for key/value counters
CREATE TABLE IF NOT EXISTS keepalive (
                                         key VARCHAR(255) PRIMARY KEY,
    value BIGINT NOT NULL
    );

-- Grant all privileges on the table to the user
GRANT ALL PRIVILEGES ON TABLE keepalive TO keepalive;

-- Initialize key/value
INSERT INTO keepalive (key, value)
VALUES ('counter', 0)
    ON CONFLICT (key) DO UPDATE
                             SET value = EXCLUDED.value;
