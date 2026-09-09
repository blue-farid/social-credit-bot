// Worker entry point: Telegram posts updates here (webhook), we dispatch by
// update type. Replaces both the Go long-polling loop and the tgcloud
// platform's automatic handler routing.

import { drizzle } from 'drizzle-orm/d1';
import { createApi } from './api.js';
import handleMessage from './handlers/message.js';

export default {
  async fetch(request, env) {
    // Anything that isn't a webhook POST — browsers, uptime checks — gets a
    // plain OK. This is the equivalent of the Go version's /health endpoint.
    if (request.method !== 'POST') {
      return new Response('ok');
    }

    // The Worker URL is public, so without this anyone who learns it could post
    // fake updates. Set with `wrangler secret put WEBHOOK_SECRET` and pass the
    // same value as `secret_token` to setWebhook (see README).
    if (env.WEBHOOK_SECRET) {
      if (request.headers.get('X-Telegram-Bot-Api-Secret-Token') !== env.WEBHOOK_SECRET) {
        return new Response('forbidden', { status: 403 });
      }
    }

    let update;
    try {
      update = await request.json();
    } catch {
      return new Response('bad request', { status: 400 });
    }

    const ctx = { db: drizzle(env.DB), api: createApi(env.BOT_TOKEN) };

    try {
      if (update.message) {
        await handleMessage(update.message, ctx);
      }
    } catch (err) {
      // Always answer 200 below: a non-2xx makes Telegram redeliver the same
      // update repeatedly, which for a logic bug means an endless retry loop.
      // Inspect failures with `npx wrangler tail`.
      console.error('handler failed', err);
    }

    return new Response('ok');
  },
};
