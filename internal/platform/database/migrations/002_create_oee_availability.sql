CREATE TABLE IF NOT EXISTS shifts (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT    NOT NULL,
    machine_id   TEXT    NOT NULL,
    start_minute INTEGER NOT NULL,
    end_minute   INTEGER NOT NULL,
    weekdays     TEXT    NOT NULL,
    timezone     TEXT    NOT NULL,
    active       INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_shifts_machine_id ON shifts(machine_id);

CREATE TABLE IF NOT EXISTS shift_breaks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    shift_id     INTEGER NOT NULL,
    start_minute INTEGER NOT NULL,
    end_minute   INTEGER NOT NULL,
    type         TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_shift_breaks_shift_id ON shift_breaks(shift_id);

CREATE TABLE IF NOT EXISTS state_intervals (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    machine_id      TEXT    NOT NULL,
    state           TEXT    NOT NULL,
    is_downtime     INTEGER NOT NULL,
    is_planned_stop INTEGER NOT NULL,
    started_at      TEXT    NOT NULL,
    ended_at        TEXT,
    reason          TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_state_intervals_machine_started ON state_intervals(machine_id, started_at);

CREATE INDEX IF NOT EXISTS idx_state_intervals_machine_ended ON state_intervals(machine_id, ended_at)
