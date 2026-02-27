/**
 * Edge middleware — pass-through (rate limiting removed; no external KV dependency).
 *
 * The previous implementation used @vercel/kv for per-IP rate limiting.
 * Rate limiting has been removed to eliminate the Upstash/external dependency.
 * The relay is low-traffic (OAuth flows only) so rate limiting is not critical.
 */

import { next } from "@vercel/edge";

export const config = {
  matcher: ["/api/:path*", "/callback", "/token/:path*", "/status/:path*"],
};

export default function middleware(_request: Request): Response {
  return next();
}
