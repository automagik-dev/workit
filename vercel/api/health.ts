/**
 * GET /health — Health check endpoint.
 *
 * Returns: { "status": "ok", "timestamp": "..." }
 */

import type { VercelRequest, VercelResponse } from "@vercel/node";

export default function handler(
  req: VercelRequest,
  res: VercelResponse,
): void {
  if (req.method !== "GET") {
    res.status(405).json({ error: "Method not allowed" });
    return;
  }

  res.status(200).json({
    status: "ok",
    timestamp: new Date().toISOString(),
  });
}
