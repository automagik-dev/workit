module.exports = {
  apps: [{
    name: 'gog-auth-server',
    script: './auth-server',
    args: '--port 8089 --credentials-file /home/genie/.config/gogcli/credentials.json --redirect-url https://gogoauth.namastex.io/callback',
    cwd: '/opt/gog-auth-server',
    interpreter: 'none',
    autorestart: true,
    max_memory_restart: '100M',
  }]
};
