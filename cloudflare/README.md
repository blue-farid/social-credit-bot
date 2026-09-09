# social-credit-bot — Cloudflare Workers port

Webhook-driven port of the Go bot, running on Cloudflare Workers with D1
(SQLite) via drizzle-orm. No server to run or patch.

This is a **sibling** of `../serverless/` (the Telegram-serverless-platform
port, waiting on waitlist access). Same game logic, different host — see
*Relationship to the other ports* below.

## Deploy

Everything below runs from this directory.

```bash
npm install
npx wrangler login                      # 1. authorize wrangler with your Cloudflare account
npx wrangler d1 create social-credit-bot # 2. create the DB, copy the printed database_id
```

3. Paste that `database_id` into `wrangler.toml` (replacing the
   `00000000-...` placeholder, which only exists so local `wrangler dev` runs).

```bash
npx wrangler d1 migrations apply social-credit-bot --remote   # 4. create the credits table
npx wrangler secret put BOT_TOKEN        # 5. paste your @BotFather bot token
npx wrangler secret put WEBHOOK_SECRET   # 6. paste any random string you generate
npx wrangler deploy                      # 7. deploy — prints your Worker URL
```

8. Point Telegram at the Worker, using the same `WEBHOOK_SECRET` from step 6:

```bash
curl "https://api.telegram.org/bot<BOT_TOKEN>/setWebhook" \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://social-credit-bot.<your-subdomain>.workers.dev/",
    "secret_token": "<WEBHOOK_SECRET>",
    "allowed_updates": ["message"]
  }'
```

curl "https://api.telegram.org/bot7039348559:AAFTCtEnuaJjDOB2tU4nbh5pmH0PH1j3O2o/setWebhook" \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://social-credit-bot.blue-farid.workers.dev/",
    "secret_token": "thisisjustasecretrealonejustHRDtounderstand",
    "allowed_updates": ["message"]
  }'

The URL is `<worker-name>.<your-account-subdomain>.workers.dev` — **not**
`<worker-name>.workers.dev`, which won't resolve and makes Telegram reject the
call with `Failed to resolve host`. `wrangler deploy` prints the exact URL on
its last-but-one line; it's also in the dashboard under Workers & Pages.

Never paste the real token or secret into this file — it's committed to git.
Keep them in the Worker secrets (`wrangler secret put`) and your password
manager only.

`WEBHOOK_SECRET` matters: the Worker URL is public, so without it anyone who
learns the URL could post fake updates. `src/index.js` rejects any POST whose
`X-Telegram-Bot-Api-Secret-Token` header doesn't match.

Verify with `curl "https://api.telegram.org/bot<BOT_TOKEN>/getWebhookInfo"`,
then message the bot. Live logs: `npm run tail`.

## Local development

```bash
npx wrangler d1 migrations apply social-credit-bot --local
npm run dev
```

`.dev.vars` (gitignored) holds local env values; a dummy `BOT_TOKEN` is enough
to exercise DB logic — Bot API calls will fail with 401, which the handler logs
and swallows. Feed it fake updates:

```bash
curl -s -X POST http://localhost:8787 -H 'Content-Type: application/json' \
  -d '{"update_id":1,"message":{"message_id":1,"chat":{"id":-100},"from":{"id":111,"username":"alice"},"text":"hi"}}'

npx wrangler d1 execute social-credit-bot --local --command "SELECT * FROM credits"
```

Note that the Worker always answers `200` even when a handler throws — a
non-2xx makes Telegram redeliver the same update in a loop. So check
`wrangler tail` / dev output for errors, not the HTTP status.

## Layout

| Path | What it is |
|---|---|
| `src/index.js` | Worker entry: verifies the webhook secret, parses the update, dispatches. Replaces the Go long-polling loop. |
| `src/handlers/message.js` | Registration, sticker-reply game mechanics, `/credits` and `/money`. |
| `src/credit.js` | DB layer (was Go's `CreditService`). `db` is passed in per request from the D1 binding. |
| `src/api.js` | ~30-line Bot API client; a Proxy so `api.<anyMethod>({...})` works. |
| `src/config.js` | Sticker IDs + initial balance (from the Go `config.yaml`). |
| `src/schema.js` | drizzle table definition. |
| `migrations/` | D1 migrations, applied by `wrangler d1 migrations apply`. |

Schema changes: edit **both** `src/schema.js` (runtime queries) and add a new
`migrations/000N_*.sql` (the DDL). They're separate sources of truth here — add
drizzle-kit if that becomes annoying.

## Relationship to the other ports

Three implementations of the same bot now live in this repo:

- `../` (Go) — original, long-polling + Postgres/SQLite. Has the activity-check feature.
- `../serverless/` — Telegram serverless platform (JS). Blocked on waitlist access.
- `./` (this) — Cloudflare Workers + D1.

`src/credit.js` and `src/handlers/message.js` are near-copies of their
`../serverless/` counterparts, with three deliberate differences:

1. **Imports**: relative paths with `.js` extensions here; bare module names
   (`from 'schema'`) on the Telegram platform. Copying code between the two
   without adjusting imports will not work.
2. **`db` is a parameter**, not a global — it comes from the Worker's D1
   binding per request.
3. **`api` is our own** `src/api.js` rather than platform-injected.

The activity-check ("are you alive?") feature is **not** ported here either,
matching `../serverless/`. Unlike the Telegram platform, Cloudflare *does* have
Cron Triggers, so it could be rebuilt — a cron that scans due timestamps in D1
would be more robust than the Go version's in-memory `scheduledJobs` map, which
lost all state on restart.

## Existing data

D1 starts empty; your current Postgres/SQLite rows are not migrated. To carry
balances over, export `credits` (`user_id`, `username`, `credit`, `money`) and
load it with `wrangler d1 execute --remote --file=dump.sql`.
