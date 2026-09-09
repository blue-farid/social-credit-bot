-- Mirrors src/schema.js. Keep the two in sync when adding columns
-- (or add drizzle-kit to generate these automatically).
CREATE TABLE IF NOT EXISTS credits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER NOT NULL UNIQUE,
  username TEXT,
  credit INTEGER NOT NULL DEFAULT 0,
  money INTEGER NOT NULL DEFAULT 0
);
