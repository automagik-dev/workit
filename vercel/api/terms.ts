/**
 * GET /terms — Terms of Service page.
 */

import type { VercelRequest, VercelResponse } from "@vercel/node";

const TERMS_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <title>Terms of Service — Automagik Auth Relay</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      max-width: 720px;
      margin: 0 auto;
      padding: 40px 20px;
      color: #333;
      line-height: 1.7;
      background: #fafafa;
    }
    h1 { color: #764ba2; border-bottom: 2px solid #764ba2; padding-bottom: 12px; }
    h2 { color: #667eea; margin-top: 32px; }
    .brand { text-align: center; margin-top: 48px; color: #999; font-size: 14px; }
    .brand a { color: #764ba2; text-decoration: none; }
    .updated { color: #999; font-size: 14px; }
  </style>
</head>
<body>
  <h1>Terms of Service</h1>
  <p class="updated">Last updated: February 2026</p>

  <h2>1. Service Description</h2>
  <p>
    The Automagik Auth Relay (<strong>auth.automagik.dev</strong>) provides a transient OAuth
    callback service for the <a href="https://github.com/automagik-dev/workit">workit</a> CLI tool.
    It facilitates Google OAuth authorization flows for headless and remote environments.
  </p>

  <h2>2. Acceptable Use</h2>
  <p>You agree to use this service only for:</p>
  <ul>
    <li>Authenticating your own Google accounts via the workit CLI or compatible tools.</li>
    <li>Legitimate OAuth authorization flows.</li>
  </ul>
  <p>You agree <strong>not</strong> to:</p>
  <ul>
    <li>Attempt to access tokens belonging to other users.</li>
    <li>Send automated requests exceeding reasonable rate limits.</li>
    <li>Use the service for any unlawful purpose.</li>
    <li>Reverse-engineer or probe the service infrastructure.</li>
  </ul>

  <h2>3. No Warranty</h2>
  <p>
    This service is provided <strong>"as is"</strong> without warranty of any kind.
    We do not guarantee uptime, availability, or fitness for a particular purpose.
  </p>

  <h2>4. Limitation of Liability</h2>
  <p>
    Namastex and its contributors shall not be liable for any damages arising from the use
    or inability to use this service, including but not limited to loss of data or unauthorized access
    resulting from factors outside our control.
  </p>

  <h2>5. Rate Limiting</h2>
  <p>
    To ensure fair access, this service enforces rate limits. Excessive requests may be
    temporarily blocked. Current limits: 20 requests per minute per IP for API endpoints.
  </p>

  <h2>6. Changes</h2>
  <p>
    We may update these terms at any time. Continued use of the service after changes
    constitutes acceptance of the new terms.
  </p>

  <h2>7. Contact</h2>
  <p>
    Questions? Reach us at <strong>legal@namastex.io</strong> or open an issue on
    <a href="https://github.com/automagik-dev/workit">GitHub</a>.
  </p>

  <div class="brand">
    <a href="https://automagik.dev">Automagik</a> · A <a href="https://namastex.io">Namastex</a> project
  </div>
</body>
</html>`;

export default function handler(
  req: VercelRequest,
  res: VercelResponse,
): void {
  if (req.method !== "GET") {
    res.status(405).send("Method not allowed");
    return;
  }

  res
    .status(200)
    .setHeader("Content-Type", "text/html; charset=utf-8")
    .send(TERMS_HTML);
}
