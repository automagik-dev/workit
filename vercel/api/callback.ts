/**
 * GET /callback — OAuth callback handler.
 *
 * Google redirects here with ?code=...&state=...
 * We exchange the code for tokens, store in Vercel KV, and show success/error HTML.
 *
 * Ported from auth-server/handlers.go handleCallback.
 */

import type { VercelRequest, VercelResponse } from "@vercel/node";
import { storeToken, type TokenEntry } from "../lib/kv.js";
import { successPage, errorPage } from "../lib/html.js";

/** Google OAuth token endpoint. */
const GOOGLE_TOKEN_URL = "https://oauth2.googleapis.com/token";

export default async function handler(
  req: VercelRequest,
  res: VercelResponse,
): Promise<void> {
  if (req.method !== "GET") {
    res.status(405).send("Method not allowed");
    return;
  }

  const code = req.query.code as string | undefined;
  const state = req.query.state as string | undefined;
  const oauthError = req.query.error as string | undefined;

  // Check for OAuth error response from Google
  if (oauthError) {
    const desc = (req.query.error_description as string) || "";
    res
      .status(400)
      .setHeader("Content-Type", "text/html; charset=utf-8")
      .send(errorPage(`OAuth error: ${oauthError} — ${desc}`));
    return;
  }

  if (!code) {
    res
      .status(400)
      .setHeader("Content-Type", "text/html; charset=utf-8")
      .send(errorPage("Missing authorization code"));
    return;
  }

  if (!state) {
    res
      .status(400)
      .setHeader("Content-Type", "text/html; charset=utf-8")
      .send(errorPage("Missing state parameter"));
    return;
  }

  // Read OAuth credentials from environment
  const clientId = process.env.WK_CLIENT_ID;
  const clientSecret = process.env.WK_CLIENT_SECRET;
  const redirectUri =
    process.env.WK_REDIRECT_URL || "https://auth.automagik.dev/callback";

  if (!clientId || !clientSecret) {
    console.error("Missing WK_CLIENT_ID or WK_CLIENT_SECRET");
    res
      .status(500)
      .setHeader("Content-Type", "text/html; charset=utf-8")
      .send(errorPage("Server misconfigured — missing OAuth credentials"));
    return;
  }

  // Exchange authorization code for tokens
  try {
    const tokenResponse = await fetch(GOOGLE_TOKEN_URL, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        code,
        client_id: clientId,
        client_secret: clientSecret,
        redirect_uri: redirectUri,
        grant_type: "authorization_code",
      }),
    });

    if (!tokenResponse.ok) {
      const errBody = await tokenResponse.text();
      console.error(
        `Token exchange failed (${tokenResponse.status}): ${errBody}`,
      );
      res
        .status(500)
        .setHeader("Content-Type", "text/html; charset=utf-8")
        .send(
          errorPage("Failed to exchange authorization code for token"),
        );
      return;
    }

    const tokens = (await tokenResponse.json()) as {
      access_token: string;
      refresh_token?: string;
      token_type: string;
      expires_in: number;
    };

    // Calculate expiry from expires_in
    const expiry = new Date(
      Date.now() + tokens.expires_in * 1000,
    ).toISOString();

    const entry: TokenEntry = {
      access_token: tokens.access_token,
      refresh_token: tokens.refresh_token || "",
      token_type: tokens.token_type || "Bearer",
      expiry,
      consumed: false,
      created_at: new Date().toISOString(),
    };

    await storeToken(state, entry);
    console.log(`Token stored for state: ${state}`);

    res
      .status(200)
      .setHeader("Content-Type", "text/html; charset=utf-8")
      .send(successPage(state));
  } catch (err) {
    console.error(`Token exchange error for state ${state}:`, err);
    res
      .status(500)
      .setHeader("Content-Type", "text/html; charset=utf-8")
      .send(errorPage("Failed to exchange authorization code for token"));
  }
}
