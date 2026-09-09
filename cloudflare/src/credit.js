// Equivalent of the Go CreditService / the tgcloud port's lib/credit.js.
// Note: unlike the tgcloud port, modules here are imported by relative path
// WITH the .js extension, and `db` is passed in per request (it comes from the
// Worker's D1 binding rather than being a global).

import { eq, desc, sql } from 'drizzle-orm';
import { credits } from './schema.js';

export async function getUserCredit(db, userId) {
  return db.select().from(credits).where(eq(credits.userId, userId)).get();
}

// Atomically creates the row if it doesn't exist yet. Returns the created row,
// or null if the user already existed (no row inserted).
export async function ensureUser(db, userId, username, initialBalance) {
  const rows = await db
    .insert(credits)
    .values({ userId, username, credit: 0, money: initialBalance })
    .onConflictDoNothing({ target: credits.userId })
    .returning();
  return rows[0] ?? null;
}

export async function updateUsername(db, userId, username) {
  await db.update(credits).set({ username }).where(eq(credits.userId, userId));
}

export async function addCredit(db, userId, amount) {
  await db
    .update(credits)
    .set({ credit: sql`${credits.credit} + ${amount}` })
    .where(eq(credits.userId, userId));
}

export async function getTopCredits(db, limit) {
  return db.select().from(credits).orderBy(desc(credits.credit)).limit(limit).all();
}

export async function getTopMoney(db, limit) {
  return db.select().from(credits).orderBy(desc(credits.money)).limit(limit).all();
}

// Throws if the sender has no money. D1 has no interactive transactions, so the
// two updates are separate atomic statements rather than one transaction.
export async function transferMoney(db, senderId, receiverId) {
  const sender = await getUserCredit(db, senderId);
  if (!sender || sender.money <= 0) {
    throw new Error('insufficient balance');
  }

  const [updatedSender] = await db
    .update(credits)
    .set({ money: sql`${credits.money} - 1` })
    .where(eq(credits.userId, senderId))
    .returning();

  const [updatedReceiver] = await db
    .update(credits)
    .set({ money: sql`${credits.money} + 1` })
    .where(eq(credits.userId, receiverId))
    .returning();

  return { sender: updatedSender, receiver: updatedReceiver };
}
