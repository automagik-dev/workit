/**
 * GET /token/{state} — Retrieve and consume an OAuth token.
 *
 * Returns:
 *   200 — Token JSON (access_token, refresh_token, token_type, expiry)
 *   202 — Pending (user hasn't completed OAuth yet)
 *   404 — Not found or expired
 *   410 — Already consumed (one-time retrieval)
 *
 * Ported from auth-server/handlers.go handleToken.
 */

import type { VercelRequest, VercelResponse } from "@vercel/node";
import { consumeToken } from "../../lib/kv.js";

export default async function handler(
  req: VercelRequest,
  res: VercelResponse,
): Promise<void> {
  if (req.method !== "GET") {
    res.status(405).json({ error: "Method not allowed" });
    return;
  }

  const state = req.query.state as string | undefined;
  if (!state) {
    res.status(400).json({ error: "Missing state parameter" });
    return;
  }

  const { entry, status } = await consumeToken(state);

  switch (status) {
    case "ready":
      res.status(200).json({
        access_token: entry!.access_token,
        refresh_token: entry!.refresh_token,
        token_type: entry!.token_type,
        expiry: entry!.expiry,
      });
      break;

    case "pending":
      res.status(202).json({
        status: "pending",
        message: "Token not yet available, please try again",
      });
      break;

    case "consumed":
      res.status(410).json({
        error: "consumed",
        message: "Token has already been retrieved",
      });
      break;

    case "not_found":
      res.status(404).json({
        error: "not_found",
        message: "Token not found or expired",
      });
      break;
  }
}
