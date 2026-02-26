/**
 * Edge middleware — rate limiting for the OAuth relay.
 *
 * Uses Vercel KV to track request counts per IP.
 * Limits: 20 req/min for /api/callback and /api/token.
 * /health, /privacy, /terms are unrestricted.
 */

import { next } from "@vercel/edge";
import { kv } from "@vercel/kv";

/** Maximum requests per window. */
const RATE_LIMIT = 20;

/** Window size in seconds. */
const WINDOW_SECONDS = 60;

/** Paths exempt from rate limiting. */
const EXEMPT_PATHS = ["/api/health", "/health", "/api/privacy", "/privacy", "/api/terms", "/terms"];

export const config = {
  matcher: ["/api/:path*", "/callback", "/token/:path*", "/status/:path*"],
};

export default async function middleware(request: Request): Promise<Response> {
  const url = new URL(request.url);

  // Skip rate limiting for exempt paths
  if (EXEMPT_PATHS.some((p) => url.pathname === p)) {
    return next();
  }

  // Get client IP from headers (Vercel provides these)
  const ip =
    request.headers.get("x-forwarded-for")?.split(",")[0]?.trim() ||
    request.headers.get("x-real-ip") ||
    "unknown";

  const key = `ratelimit:${ip}`;

  try {
    // Increment counter; set TTL on first request in window
    const count = await kv.incr(key);
    if (count === 1) {
      await kv.expire(key, WINDOW_SECONDS);
    }

    // Add rate limit headers
    const headers = new Headers();
    headers.set("X-RateLimit-Limit", String(RATE_LIMIT));
    headers.set("X-RateLimit-Remaining", String(Math.max(0, RATE_LIMIT - count)));

    if (count > RATE_LIMIT) {
      headers.set("Retry-After", String(WINDOW_SECONDS));
      return new Response(
        JSON.stringify({
          error: "rate_limited",
          message: `Too many requests. Limit: ${RATE_LIMIT} per ${WINDOW_SECONDS}s.`,
        }),
        {
          status: 429,
          headers: {
            "Content-Type": "application/json",
            ...Object.fromEntries(headers),
          },
        },
      );
    }

    // Pass through with rate limit headers
    const response = next();
    // Edge middleware can't easily add headers to next() responses in all runtimes,
    // but the headers are informational. The blocking at 429 is what matters.
    return response;
  } catch (err) {
    // If KV is down, fail open (allow the request)
    console.error("Rate limit check failed, allowing request:", err);
    return next();
  }
}
