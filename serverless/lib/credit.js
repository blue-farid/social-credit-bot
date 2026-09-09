import { db } from 'sdk';
import { eq, desc, sql } from 'sdk/db';
import { credits } from 'schema';

export async function getUserCredit(userId) {
  return db.select().from(credits).where(eq(credits.userId, userId)).get();
}

// Atomically creates the row if it doesn't exist yet. Returns the created row,
// or null if the user already existed (no row inserted).
export async function ensureUser(userId, username, initialBalance) {
  const rows = await db.insert(credits)
    .values({ userId, username, credit: 0, money: initialBalance })
    .onConflictDoNothing({ target: credits.userId })
    .returning()
    .run();
  return rows[0] ?? null;
}

export async function updateUsername(userId, username) {
  await db.update(credits).set({ username }).where(eq(credits.userId, userId)).run();
}

export async function addCredit(userId, amount) {
  await db.update(credits)
    .set({ credit: sql`${credits.credit} + ${amount}` })
    .where(eq(credits.userId, userId))
    .run();
}

export async function getTopCredits(limit) {
  return db.select().from(credits).orderBy(desc(credits.credit)).limit(limit).all();
}

export async function getTopMoney(limit) {
  return db.select().from(credits).orderBy(desc(credits.money)).limit(limit).all();
}

// Throws if the sender has no money. Not atomic across the two rows — the
// platform has no db.transaction() — but each individual update is.
export async function transferMoney(senderId, receiverId) {
  const sender = await getUserCredit(senderId);
  if (!sender || sender.money <= 0) {
    throw new Error('insufficient balance');
  }

  const [updatedSender] = await db.update(credits)
    .set({ money: sql`${credits.money} - 1` })
    .where(eq(credits.userId, senderId))
    .returning()
    .run();

  const [updatedReceiver] = await db.update(credits)
    .set({ money: sql`${credits.money} + 1` })
    .where(eq(credits.userId, receiverId))
    .returning()
    .run();

  return { sender: updatedSender, receiver: updatedReceiver };
}
