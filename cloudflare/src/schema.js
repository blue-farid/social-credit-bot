import { sqliteTable, integer, text } from 'drizzle-orm/sqlite-core';

// DDL lives in migrations/0001_create_credits.sql — keep both in sync.
export const credits = sqliteTable('credits', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  userId: integer('user_id').notNull().unique(),
  username: text('username'),
  credit: integer('credit').notNull().default(0),
  money: integer('money').notNull().default(0),
});
