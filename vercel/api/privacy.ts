/**
 * GET /privacy — Privacy policy page.
 */

import type { VercelRequest, VercelResponse } from "@vercel/node";

const PRIVACY_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <title>Privacy Policy — Automagik Auth Relay</title>
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
  <h1>Privacy Policy</h1>
  <p class="updated">Last updated: February 2026</p>

  <h2>What This Service Does</h2>
  <p>
    The Automagik Auth Relay (<strong>auth.automagik.dev</strong>) is a lightweight OAuth callback
    proxy. It receives authorization codes from Google's OAuth flow, exchanges them for access tokens,
    and holds the tokens temporarily so your CLI tool can retrieve them.
  </p>

  <h2>Data We Collect</h2>
  <p>This service processes the following data <strong>transiently</strong>:</p>
  <ul>
    <li><strong>OAuth tokens</strong> — stored in encrypted memory (Vercel KV) for a maximum of 5 minutes, then automatically deleted.</li>
    <li><strong>State parameter</strong> — a random string used to match the callback to the CLI request. Deleted with the token.</li>
  </ul>
  <p>We do <strong>not</strong> store, log, or retain:</p>
  <ul>
    <li>Your email address</li>
    <li>Your Google account data</li>
    <li>Your IP address</li>
    <li>Any cookies or tracking identifiers</li>
  </ul>

  <h2>Data Retention</h2>
  <p>
    All token data is automatically purged after <strong>5 minutes</strong> (300 seconds TTL).
    Tokens are also marked consumed after a single retrieval and cannot be read again.
  </p>

  <h2>Third Parties</h2>
  <p>
    This service is hosted on <a href="https://vercel.com/legal/privacy-policy">Vercel</a>.
    Token exchange is performed with <a href="https://policies.google.com/privacy">Google OAuth</a>.
    No other third parties receive your data.
  </p>

  <h2>Google API Services</h2>
  <p>
    This application's use and transfer of information received from Google APIs adheres to the
    <a href="https://developers.google.com/terms/api-services-user-data-policy">
    Google API Services User Data Policy</a>, including the Limited Use requirements.
  </p>

  <h2>Contact</h2>
  <p>
    Questions? Reach us at <strong>privacy@namastex.io</strong> or open an issue on
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
    .send(PRIVACY_HTML);
}
