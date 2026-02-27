/**
 * GET /status/{state} — Check token status without consuming.
 *
 * Returns: { "status": "ready" | "pending" | "consumed" | "not_found" }
 *
 * Ported from auth-server/handlers.go handleStatus.
 */

import type { VercelRequest, VercelResponse } from "@vercel/node";
import { checkTokenStatus } from "../../lib/kv.js";

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

  const status = await checkTokenStatus(state);
  res.status(200).json({ status });
}
