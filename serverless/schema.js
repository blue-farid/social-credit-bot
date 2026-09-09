import { table, integer, text } from 'sdk/db';

export const credits = table('credits', {
  id: integer('id').primaryKey({ autoIncrement: true }),
  userId: integer('user_id').notNull().unique(),
  username: text('username'),
  credit: integer('credit').notNull().default(0),
  money: integer('money').notNull().default(0),
});
