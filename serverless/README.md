# social-credit-bot — Telegram Serverless port

Ported from the Go bot in the repo root to Telegram's serverless platform
(`@tgcloud/cli`). See `AGENTS.md` and `docs/tgcloud-sdk.md` for how the
platform works.

## What changed vs. the Go version

- **Language/runtime**: Go + long polling + Postgres/SQLite (gorm) → JavaScript
  handlers in a V8 isolate, with the platform's own SQLite-backed DB. There is
  no external Postgres connection.
- **Activity-check feature removed.** The old `ActivityService` sent a
  scheduled "are you alive?" ping (cron) and ran per-user delayed retry timers.
  The serverless platform only runs code in response to an incoming Telegram
  update — there is no cron or delayed-execution primitive — so this feature
  has no direct equivalent here. Dropped along with it: the `alive_score`
  column, the "alive" inline button/callback handling, and the `/alive`
  leaderboard command. If you want it back, it would need to live outside this
  platform (e.g. an external cron hitting a webhook you host yourself), which
  defeats the point of using this platform.
- Health-check HTTP endpoint (`internal/handlers/health_handler.go`) isn't
  ported — nothing to health-check on a managed serverless platform.
- Sticker IDs and `initial_balance` from `config.yaml` are now constants in
  `lib/config.js` (edit directly — there's no env var system here beyond the
  bot token).

## Existing data

This does **not** migrate your existing Postgres/SQLite rows (`credits`,
activity tables). The platform's DB starts empty. If you want to carry over
balances, export the `credits` table (`user_id`, `username`, `credit`,
`money`) from the old DB and load it in with `db.insert(credits).values([...])`
via a one-off `tgcloud run`, or ask for a script to do this.

## Setup

```bash
cd serverless
npm install
npx tgcloud login    # paste the CLI access token from @BotFather (app<id>:<secret>)
npx tgcloud push     # deploy handlers/lib/schema
npx tgcloud migrate  # create the `credits` table
```

`push` manages the webhook for you — no manual `setWebhook` call needed.

## Local testing (no deploy)

```bash
npx tgcloud run handlers/message '{ "chat": { "id": 1 }, "from": { "id": 1, "username": "test" }, "text": "hi" }'
```
