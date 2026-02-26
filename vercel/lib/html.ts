/**
 * HTML page templates for the OAuth relay.
 * Ported from auth-server/handlers.go renderSuccessPage / renderErrorPage.
 */

const baseStyle = `
  body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
    margin: 0;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  }
  .container {
    background: white;
    padding: 40px;
    border-radius: 16px;
    box-shadow: 0 10px 40px rgba(0,0,0,0.2);
    text-align: center;
    max-width: 400px;
  }
  .icon { font-size: 64px; margin-bottom: 20px; }
  p { color: #666; line-height: 1.6; }
  .brand {
    margin-top: 32px;
    font-size: 12px;
    color: #999;
  }
  .brand a { color: #764ba2; text-decoration: none; }
`;

export function successPage(state: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <title>Authorization Successful</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>${baseStyle}
    h1 { color: #22c55e; margin-bottom: 16px; }
    .state {
      font-family: monospace;
      background: #f3f4f6;
      padding: 8px 12px;
      border-radius: 6px;
      font-size: 14px;
      word-break: break-all;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="icon">&#x2705;</div>
    <h1>Authorization Successful</h1>
    <p>You have successfully authorized the application.</p>
    <p>You can close this window and return to your terminal.</p>
    <p style="margin-top: 24px; font-size: 12px; color: #999;">
      State: <span class="state">${escapeHtml(state)}</span>
    </p>
    <div class="brand">Powered by <a href="https://automagik.dev">Automagik</a></div>
  </div>
</body>
</html>`;
}

export function errorPage(message: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <title>Authorization Failed</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>${baseStyle}
    h1 { color: #ef4444; margin-bottom: 16px; }
    .error-message {
      background: #fef2f2;
      color: #b91c1c;
      padding: 12px;
      border-radius: 8px;
      margin-top: 16px;
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="icon">&#x274C;</div>
    <h1>Authorization Failed</h1>
    <p>There was a problem completing the authorization.</p>
    <div class="error-message">${escapeHtml(message)}</div>
    <p style="margin-top: 24px;">Please try again or contact support.</p>
    <div class="brand">Powered by <a href="https://automagik.dev">Automagik</a></div>
  </div>
</body>
</html>`;
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
