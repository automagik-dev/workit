/**
 * Vercel KV helpers for OAuth token storage.
 *
 * Token lifecycle:
 *   1. Google redirects to /callback with code + state
 *   2. /callback exchanges code → tokens, stores in KV keyed by state (TTL 5min)
 *   3. CLI polls /token/{state} — returns token and marks consumed
 *
 * KV key schema:
 *   token:{state}   → JSON TokenEntry   (EX 300)
 */

import { kv } from "@vercel/kv";

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
  await kv.set(tokenKey(state), JSON.stringify(entry), {
    ex: TOKEN_TTL_SECONDS,
  });
}

/** Get a token entry without consuming it. */
export async function getTokenEntry(
  state: string,
): Promise<TokenEntry | null> {
  const raw = await kv.get<string>(tokenKey(state));
  if (!raw) return null;
  // kv.get with a type may already parse JSON; handle both cases
  if (typeof raw === "string") {
    return JSON.parse(raw) as TokenEntry;
  }
  return raw as unknown as TokenEntry;
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
    await kv.set(tokenKey(state), JSON.stringify(entry), { ex: ttl });
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
