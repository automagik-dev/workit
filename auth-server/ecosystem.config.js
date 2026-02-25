module.exports = {
  apps: [{
    name: 'wk-auth-server',
    script: './auth-server',
    args: '--port 8089 --credentials-file /home/genie/.config/workit/credentials.json --redirect-url https://wkauth.namastex.io/callback',
    cwd: '/opt/wk-auth-server',
    interpreter: 'none',
    autorestart: true,
    max_memory_restart: '100M',
  }]
};
