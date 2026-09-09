// Ported from internal/handlers/message_handler.go (Go) via serverless/handlers/message.js.
// The "alive" activity-check feature is not included — see cloudflare/README.md.

import { INITIAL_BALANCE, STICKERS } from '../config.js';
import * as Credit from '../credit.js';

export default async function handleMessage(message, { db, api }) {
  if (message.from) {
    await trackUser(message, { db, api });
  }

  if (message.reply_to_message && message.sticker) {
    await handleStickerReply(message, { db, api });
    return;
  }

  const command = getCommand(message);
  if (command) {
    await handleCommand(command, message, { db, api });
  }
}

async function trackUser(message, { db, api }) {
  const userId = message.from.id;
  const username = message.from.username;

  const existing = await Credit.getUserCredit(db, userId);
  if (!existing) {
    await Credit.ensureUser(db, userId, username, INITIAL_BALANCE);
    await api.sendMessage({
      chat_id: message.chat.id,
      text: `💰 Welcome @${username}! You received ${INITIAL_BALANCE} initial money.`,
    });
  } else if (existing.username !== username) {
    await Credit.updateUsername(db, userId, username);
  }
}

async function handleStickerReply(message, { db, api }) {
  if (message.from.id === message.reply_to_message.from.id) {
    if (await handleSelfReplyFraud(message, { db, api })) return;
  }

  if (STICKERS.transfer.includes(message.sticker.file_unique_id)) {
    await handleMoneyTransfer(message, { db, api });
    return;
  }

  await handleSocialCredit(message, { db, api });
}

async function handleSelfReplyFraud(message, { db, api }) {
  if (stickerType(message.sticker.file_unique_id) !== 'positive') return false;

  const cheater = await Credit.getUserCredit(db, message.from.id);
  if (!cheater) return false;

  await Credit.addCredit(db, message.from.id, -3);
  await api.sendMessage({
    chat_id: message.chat.id,
    text: `🚫 Fraud detected! @${cheater.username} tried to cheat by replying to their own message with a positive sticker.\nPenalty: -3 SocialCredit\nCurrent balance: ${cheater.credit - 3}`,
  });
  return true;
}

async function handleMoneyTransfer(message, { db, api }) {
  let transfer;
  try {
    transfer = await Credit.transferMoney(db, message.from.id, message.reply_to_message.from.id);
  } catch {
    await api.sendMessage({
      chat_id: message.chat.id,
      text: "❌ You don't have any money to transfer!",
    });
    return;
  }

  const { sender, receiver } = transfer;
  await api.sendMessage({
    chat_id: message.chat.id,
    text: `💰 Money Transfer:\n@${sender.username} sent 1 money to @${receiver.username}\n\n@${sender.username}'s balance: ${sender.money}\n@${receiver.username}'s balance: ${receiver.money}`,
  });
}

async function handleSocialCredit(message, { db, api }) {
  const type = stickerType(message.sticker.file_unique_id);
  if (!type) return;

  const amount = type === 'negative' ? -1 : 1;
  const targetId = message.reply_to_message.from.id;

  const user = await Credit.getUserCredit(db, targetId);
  if (!user) return;

  await Credit.addCredit(db, targetId, amount);
  await api.sendMessage({
    chat_id: message.chat.id,
    text: `@${user.username} got ${amount > 0 ? `+${amount}` : amount} SocialCredit! Total: ${user.credit + amount}`,
  });
}

function stickerType(fileUniqueId) {
  if (STICKERS.positive.includes(fileUniqueId)) return 'positive';
  if (STICKERS.negative.includes(fileUniqueId)) return 'negative';
  return null;
}

function getCommand(message) {
  const entity = message.entities?.find((e) => e.type === 'bot_command' && e.offset === 0);
  if (!entity) return null;
  return message.text.slice(entity.offset + 1, entity.offset + entity.length).split('@')[0];
}

async function handleCommand(command, message, { db, api }) {
  switch (command) {
    case 'credits':
      return handleCreditsCommand(message, { db, api });
    case 'money':
      return handleMoneyCommand(message, { db, api });
  }
}

async function handleCreditsCommand(message, { db, api }) {
  const top = await Credit.getTopCredits(db, 10);
  const text = '🌟 SocialCredit Leaderboard:\n' + top.map((c) => `@${c.username} — ${c.credit}`).join('\n');
  await api.sendMessage({ chat_id: message.chat.id, text });
}

async function handleMoneyCommand(message, { db, api }) {
  const top = await Credit.getTopMoney(db, 10);
  const text = '💰 Money Leaderboard:\n' + top.map((c) => `@${c.username} — ${c.money}`).join('\n');
  await api.sendMessage({ chat_id: message.chat.id, text });
}
