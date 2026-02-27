/**
 * In-memory token store with TTL (no external dependencies).
 *
 * Token lifecycle:
 *   1. Google redirects to /callback with code + state
 *   2. /callback exchanges code → tokens, stores in KV keyed by state (TTL 5min)
 *   3. CLI polls /token/{state} — returns token and marks consumed
 *
 * KV key schema:
 *   token:{state}   → JSON TokenEntry   (TTL 300s)
 */

// In-memory token store with TTL (no external dependencies)
const store = new Map<string, { value: string; expiresAt: number }>();

export const kv = {
  async set(key: string, value: string, ttlSeconds: number): Promise<void> {
    store.set(key, { value, expiresAt: Date.now() + ttlSeconds * 1000 });
  },
  async get(key: string): Promise<string | null> {
    const entry = store.get(key);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      store.delete(key);
      return null;
    }
    return entry.value;
  },
  async del(key: string): Promise<void> {
    store.delete(key);
  },
  async ttl(key: string): Promise<number> {
    const entry = store.get(key);
    if (!entry) return -2;
    const remaining = Math.floor((entry.expiresAt - Date.now()) / 1000);
    return remaining > 0 ? remaining : -2;
  },
};

/** TTL for stored tokens in seconds (5 minutes). */
export const TOKEN_TTL_SECONDS = 300;

/** Matches the Go TokenEntry structure. */
export interface TokenEntry {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expiry: string;
  consumed: boolean;
  created_at: string;
}

export type TokenStatus = "ready" | "pending" | "consumed" | "not_found";

function tokenKey(state: string): string {
  return `token:${state}`;
}

/** Store a token entry in KV with TTL. */
export async function storeToken(
  state: string,
  entry: TokenEntry,
): Promise<void> {
  await kv.set(tokenKey(state), JSON.stringify(entry), TOKEN_TTL_SECONDS);
}

/** Get a token entry without consuming it. */
export async function getTokenEntry(
  state: string,
): Promise<TokenEntry | null> {
  const raw = await kv.get(tokenKey(state));
  if (!raw) return null;
  return JSON.parse(raw) as TokenEntry;
}

/** Get and consume a token (one-time retrieval). */
export async function consumeToken(
  state: string,
): Promise<{ entry: TokenEntry | null; status: TokenStatus }> {
  const entry = await getTokenEntry(state);

  if (!entry) {
    return { entry: null, status: "not_found" };
  }

  if (entry.consumed) {
    return { entry: null, status: "consumed" };
  }

  if (!entry.access_token) {
    return { entry: null, status: "pending" };
  }

  // Mark consumed and update in KV (preserve remaining TTL)
  entry.consumed = true;
  const ttl = await kv.ttl(tokenKey(state));
  if (ttl > 0) {
    await kv.set(tokenKey(state), JSON.stringify(entry), ttl);
  }

  return { entry, status: "ready" };
}

/** Check token status without consuming. */
export async function checkTokenStatus(state: string): Promise<TokenStatus> {
  const entry = await getTokenEntry(state);

  if (!entry) return "not_found";
  if (entry.consumed) return "consumed";
  if (!entry.access_token) return "pending";
  return "ready";
}
