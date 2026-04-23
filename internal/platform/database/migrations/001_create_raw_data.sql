CREATE TABLE IF NOT EXISTS raw_data (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    client     TEXT    NOT NULL,
    topic      TEXT    NOT NULL,
    timezone   TEXT    NOT NULL,
    timestamp  TEXT    NOT NULL,
    payload    TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_raw_data_topic ON raw_data(topic);
CREATE INDEX IF NOT EXISTS idx_raw_data_timestamp ON raw_data(timestamp);
CREATE INDEX IF NOT EXISTS idx_raw_data_client ON raw_data(client);
