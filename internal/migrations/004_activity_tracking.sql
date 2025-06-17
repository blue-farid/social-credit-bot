-- +goose Up
CREATE TABLE IF NOT EXISTS activity_status (
    user_id BIGINT PRIMARY KEY,
    username TEXT NOT NULL,
    last_check TIMESTAMP NOT NULL,
    last_response TIMESTAMP,
    retry_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    message_id INTEGER,
    next_check_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS activity_checks (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    username TEXT NOT NULL,
    check_time TIMESTAMP NOT NULL,
    response BOOLEAN NOT NULL,
    score INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activity_checks_user_id ON activity_checks(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_checks_check_time ON activity_checks(check_time);

-- +goose Down
DROP TABLE IF EXISTS activity_checks;
DROP TABLE IF EXISTS activity_status; 