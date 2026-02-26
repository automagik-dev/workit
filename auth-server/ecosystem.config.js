const WK_AUTH_SERVER_HOME = process.env.WK_AUTH_SERVER_HOME || '/opt/wk-auth-server';
const WK_AUTH_SERVER_BIN = process.env.WK_AUTH_SERVER_BIN || `${WK_AUTH_SERVER_HOME}/auth-server`;
const WK_AUTH_SERVER_PORT = process.env.WK_AUTH_SERVER_PORT || '8089';
const WK_REDIRECT_URL = process.env.WK_REDIRECT_URL || 'https://auth.example.com/callback';
const WK_CREDENTIALS_FILE = process.env.WK_CREDENTIALS_FILE || '';

const args = [
  '--port',
  WK_AUTH_SERVER_PORT,
  '--redirect-url',
  WK_REDIRECT_URL,
];

if (WK_CREDENTIALS_FILE) {
  args.push('--credentials-file', WK_CREDENTIALS_FILE);
}

module.exports = {
  apps: [{
    name: 'wk-auth-server',
    script: WK_AUTH_SERVER_BIN,
    cwd: WK_AUTH_SERVER_HOME,
    args,
    interpreter: 'none',
    autorestart: true,
    exp_backoff_restart_delay: 1000,
    max_memory_restart: '100M',
  }],
};
