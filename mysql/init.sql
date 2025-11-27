-- Drop and recreate the database
DROP
DATABASE IF EXISTS keepalive;
CREATE
DATABASE keepalive;

-- Create user if not exists
CREATE
USER IF NOT EXISTS 'keepalive'@'%' IDENTIFIED BY 'keepalive';

-- Grant full permissions on this database
GRANT ALL PRIVILEGES ON keepalive.* TO
'keepalive'@'%';

FLUSH
PRIVILEGES;

-- Create the table for key/value counters
USE
keepalive;

CREATE TABLE IF NOT EXISTS keepalive
(
    `key`
    VARCHAR
(
    255
) PRIMARY KEY,
    `value` BIGINT NOT NULL
    );

-- Initialize key/value
INSERT INTO keepalive (`key`, `value`)
VALUES ('counter', 0) ON DUPLICATE KEY
UPDATE `value` = 0;
