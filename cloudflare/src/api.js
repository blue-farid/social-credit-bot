// Minimal Telegram Bot API client. Mirrors the shape of the tgcloud platform's
// injected `api` object (`api.sendMessage({...})`, unwrapped result, throws on
// failure) so handler code reads the same in both ports.

const BASE = 'https://api.telegram.org';

export class BotApiError extends Error {
  constructor(method, code, description) {
    super(`${method} failed: ${code} ${description}`);
    this.name = 'BotApiError';
    this.method = method;
    this.code = code;
    this.description = description;
  }
}

// A Proxy dispatches any method name, so every current and future Bot API
// method works without touching this file.
export function createApi(token) {
  return new Proxy(
    {},
    {
      get(_target, method) {
        return async (params = {}) => {
          const res = await fetch(`${BASE}/bot${token}/${method}`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(params),
          });
          const body = await res.json();
          if (!body.ok) {
            throw new BotApiError(method, body.error_code, body.description);
          }
          return body.result;
        };
      },
    },
  );
}
