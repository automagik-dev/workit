# Hapvida M365 Auth Server — Deploy Manager Handoff

**Date:** 2026-06-02  
**Owner:** Workit / KHAW integration  
**Audience:** Hapvida deploy manager / OCI HV platform team  
**Status:** PR `automagik-dev/workit#47` merged into `main`  
**Purpose:** deploy a dedicated Workit auth server inside Hapvida-controlled infrastructure so Hapvida users can safely authorize Microsoft 365 read-only access for KHAW/Workit.

---

## 1. Executive summary

We are **not** using the shared public callback relay (`auth.automagik.dev`) for Hapvida Microsoft 365.

For the Hapvida rollout, deploy a **dedicated Workit auth server** as a pod/service inside the Hapvida OCI HV environment, behind a Hapvida-controlled HTTPS ingress.

This server will be the Microsoft OAuth broker for Hapvida users:

1. KHAW/Workit creates a short-lived one-click login session on the broker.
2. The user opens the generated link.
3. Microsoft redirects back to the Hapvida-controlled callback URL.
4. The broker validates the Microsoft account and stores the token in-memory for one-time retrieval by Workit.
5. KHAW can then read authorized Microsoft 365 data for the user.

Initial pilot scope is **read-only**:

- `User.Read`
- `Mail.Read`
- `Calendars.Read`
- `offline_access`

No write scopes are requested.

This enables a safe future path where **every Hapvida user can authorize email/calendar access for KHAW**, without sending corporate OAuth tokens through Automagik's public auth domain.

---

## 2. What changed in Workit

Merged PR:

- `https://github.com/automagik-dev/workit/pull/47`

The Workit repo now contains a deployable auth server under:

```text
auth-server/
```

Important files:

```text
auth-server/Dockerfile
auth-server/main.go
auth-server/m365.go
auth-server/README.md
internal/cmd/auth_m365_link.go
```

New server endpoints:

```text
GET  /health
POST /m365/sessions
GET  /m365/start/{state}
GET  /m365/callback
GET  /status/{state}
GET  /token/{state}
```

Security hardening included:

- `/m365/sessions` requires an admin bearer token.
- state is short-lived and one-time-use.
- OAuth uses PKCE S256.
- code verifier is never returned in JSON.
- callback validates Microsoft Graph `/me` account identity.
- e-mail mismatch errors avoid logging PII.
- session-create request body has size limit.
- expired sessions are pruned.
- server can run M365-only, without Google credentials.
- Docker runtime is distroless/non-root.

---

## 3. Target deployment shape

### Recommended public URL

Choose a Hapvida-controlled HTTPS hostname, for example:

```text
https://auth-ai.hapvida.com.br
```

The exact name can be different, but it must be:

- HTTPS
- public/reachable by Microsoft Entra during OAuth redirect
- controlled by Hapvida/HV
- routed to the Workit auth server pod

### Required Microsoft redirect URI

Once the hostname is selected, configure this exact redirect URI in the Microsoft Entra App Registration:

```text
https://<hapvida-auth-domain>/m365/callback
```

Example:

```text
https://auth-ai.hapvida.com.br/m365/callback
```

This URI must match exactly. Trailing slash differences matter.

---

## 4. Required runtime environment variables

Set these on the OCI/Kubernetes deployment:

```bash
WK_PUBLIC_BASE_URL=https://<hapvida-auth-domain>
WK_M365_CLIENT_ID=<microsoft-entra-application-client-id>
WK_M365_TENANT_ID=<hapvida-microsoft-tenant-id>
WK_M365_BROKER_TOKEN=<strong-random-admin-token>
```

### Variable meaning

#### `WK_PUBLIC_BASE_URL`

Public HTTPS base URL of the deployed broker.

Example:

```bash
WK_PUBLIC_BASE_URL=https://auth-ai.hapvida.com.br
```

The server derives the callback URL as:

```text
${WK_PUBLIC_BASE_URL}/m365/callback
```

#### `WK_M365_CLIENT_ID`

Microsoft Entra **Application (client) ID** of the Hapvida-owned App Registration.

This is not a secret.

#### `WK_M365_TENANT_ID`

Microsoft Entra **Directory (tenant) ID** for Hapvida.

This is not a secret.

#### `WK_M365_BROKER_TOKEN`

Strong random bearer token used by trusted Workit/KHAW callers to create `/m365/sessions`.

This is a secret.

Generate it with a secure random generator, for example:

```bash
openssl rand -base64 48
```

Store it as a Kubernetes/OCI secret, not as plain text in manifests.

---

## 5. Docker build

Build from the `auth-server` directory:

```bash
cd auth-server
docker build -t <registry>/workit-auth-server:<tag> .
```

The Dockerfile is multi-stage and distroless:

```text
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS build
FROM gcr.io/distroless/static-debian12:nonroot
```

The image exposes port:

```text
8080
```

Entrypoint:

```text
/app/workit-auth-server
```

---

## 6. Kubernetes/OCI deployment example

Use this as a template. Replace placeholders before applying.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: workit-auth-server
  namespace: hapvida-ai
spec:
  replicas: 2
  selector:
    matchLabels:
      app: workit-auth-server
  template:
    metadata:
      labels:
        app: workit-auth-server
    spec:
      containers:
        - name: workit-auth-server
          image: <registry>/workit-auth-server:<tag>
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 8080
              name: http
          env:
            - name: WK_PUBLIC_BASE_URL
              value: "https://<hapvida-auth-domain>"
            - name: WK_M365_CLIENT_ID
              valueFrom:
                secretKeyRef:
                  name: workit-m365-auth
                  key: client-id
            - name: WK_M365_TENANT_ID
              valueFrom:
                secretKeyRef:
                  name: workit-m365-auth
                  key: tenant-id
            - name: WK_M365_BROKER_TOKEN
              valueFrom:
                secretKeyRef:
                  name: workit-m365-auth
                  key: broker-token
          readinessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 10
            periodSeconds: 30
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 500m
              memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: workit-auth-server
  namespace: hapvida-ai
spec:
  selector:
    app: workit-auth-server
  ports:
    - name: http
      port: 80
      targetPort: http
```

Ingress/load balancer must terminate HTTPS and route:

```text
https://<hapvida-auth-domain>/* -> workit-auth-server:80
```

At minimum, these paths must be reachable externally:

```text
/health
/m365/start/{state}
/m365/callback
```

`/m365/sessions`, `/token/{state}`, and `/status/{state}` should be restricted where possible to trusted networks/callers, but `/m365/sessions` is also protected by bearer token.

---

## 7. Microsoft Entra App Registration setup

Create or use a Hapvida-owned Microsoft Entra App Registration.

Portal URL:

```text
https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
```

Alternative Azure portal path:

```text
https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
```

Required settings:

### App type

Recommended:

```text
Single tenant
```

Supported account types:

```text
Accounts in this organizational directory only
```

### Redirect URI

Platform type:

```text
Web
```

Redirect URI:

```text
https://<hapvida-auth-domain>/m365/callback
```

### API permissions

Microsoft Graph delegated permissions:

```text
User.Read
Mail.Read
Calendars.Read
offline_access
```

No write permissions for pilot.

Do **not** add:

```text
Mail.Send
Mail.ReadWrite
Calendars.ReadWrite
Files.ReadWrite
```

### Values needed by deployment

From the App Registration overview page, copy:

```text
Application (client) ID
Directory (tenant) ID
```

These become:

```bash
WK_M365_CLIENT_ID=<Application client ID>
WK_M365_TENANT_ID=<Directory tenant ID>
```

---

## 8. Exact request to Bernardo

Bernardo, para gerar o link de autenticação real, precisamos destes dados do App Registration da Hapvida no Microsoft Entra:

1. **Application (client) ID**
2. **Directory (tenant) ID**
3. Confirmação de que o redirect URI abaixo foi cadastrado como `Web`:

```text
https://<domínio-HV-que-o-deploy-manager-definir>/m365/callback
```

Você consegue pegar esses dados aqui:

```text
https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
```

Caminho:

```text
Microsoft Entra admin center
→ Applications
→ App registrations
→ selecionar/criar o app da Hapvida para Workit/KHAW
→ Overview
→ copiar Application (client) ID e Directory (tenant) ID
```

E para configurar o callback:

```text
App registrations
→ selecionar o app
→ Authentication
→ Add a platform
→ Web
→ Redirect URI: https://<domínio-HV>/m365/callback
```

Importante: não precisamos de senha pessoal nem token de usuário.  
Precisamos apenas dos IDs do App Registration e do domínio final onde o broker será publicado.

O deploy manager cria o `WK_M365_BROKER_TOKEN` internamente como segredo do ambiente.

---

## 9. How to generate a user login link after deployment

Once the pod is deployed and reachable:

```bash
WK_CALLBACK_SERVER=https://<hapvida-auth-domain> \
WK_M365_BROKER_TOKEN=<same-secret-as-server> \
wk auth m365 login-link bernardo@hapvida.com.br
```

Expected output:

```text
login_url https://<hapvida-auth-domain>/m365/start/<state>
expected_email bernardo@hapvida.com.br
expires_at <timestamp>
```

Send `login_url` to the user.

The user clicks it, authenticates with Microsoft, and the broker validates that the signed-in Microsoft account matches the expected e-mail.

---

## 10. Validation checklist after deployment

### Health check

```bash
curl -fsS https://<hapvida-auth-domain>/health
```

Expected response:

```json
{"status":"ok","timestamp":"..."}
```

### Session creation must reject unauthenticated callers

```bash
curl -i -X POST https://<hapvida-auth-domain>/m365/sessions \
  -H 'Content-Type: application/json' \
  -d '{"expected_email":"bernardo@hapvida.com.br"}'
```

Expected:

```text
401 Unauthorized
```

### Session creation with bearer token

```bash
curl -fsS -X POST https://<hapvida-auth-domain>/m365/sessions \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer <broker-token>" \
  -d '{"expected_email":"bernardo@hapvida.com.br"}'
```

Expected JSON includes:

```json
{
  "state": "...",
  "expected_email": "bernardo@hapvida.com.br",
  "login_url": "https://<hapvida-auth-domain>/m365/start/...",
  "expires_at": "..."
}
```

It must **not** include:

```text
code_verifier
access_token
refresh_token
```

### Start URL redirect

Open:

```text
https://<hapvida-auth-domain>/m365/start/<state>
```

Expected:

- HTTP 302 to `login.microsoftonline.com`
- `redirect_uri` points to `https://<hapvida-auth-domain>/m365/callback`
- scopes include read-only Graph permissions
- scopes do not include write permissions

---

## 11. Operational notes

### Token storage

The current auth server stores OAuth tokens in memory for short-lived retrieval.

Implications:

- restart clears pending sessions/tokens;
- use at least two replicas only if the ingress supports sticky routing or if session state is externalized later;
- for the first pilot, one replica may be simpler unless the platform provides sticky sessions.

If using multiple replicas without sticky sessions, `/m365/start`, `/m365/callback`, and `/token/{state}` may hit different pods. The safest first deployment is:

```yaml
replicas: 1
```

or configure sticky sessions at ingress/load balancer.

### Network restrictions

Recommended:

- expose `/m365/start/*` and `/m365/callback` publicly over HTTPS;
- restrict `/m365/sessions`, `/token/*`, and `/status/*` to trusted networks where possible;
- always require `WK_M365_BROKER_TOKEN` for `/m365/sessions`.

### Logging

Do not log:

- access tokens;
- refresh tokens;
- raw e-mail bodies;
- attachment contents;
- broker token.

The current code avoids returning PKCE verifier and avoids PII in e-mail mismatch errors.

---

## 12. Acceptance criteria

Deployment is ready when:

- `GET /health` returns OK over HTTPS.
- Entra App Registration has exact redirect URI.
- `POST /m365/sessions` without bearer token returns 401.
- `POST /m365/sessions` with bearer token returns login URL.
- Login URL redirects to Microsoft.
- Microsoft redirects back to `/m365/callback` without redirect mismatch.
- Generated Microsoft authorize URL requests only read scopes.
- Bernardo can click the link and complete Microsoft login.
- Workit/KHAW can retrieve the token once via `/token/{state}`.

---

## 13. Minimal values to send back to Workit/KHAW team

Deploy manager should return:

```text
WK_PUBLIC_BASE_URL=https://<hapvida-auth-domain>
WK_M365_CLIENT_ID=<client-id>
WK_M365_TENANT_ID=<tenant-id>
```

Do not send `WK_M365_BROKER_TOKEN` in chat unless using an approved secure channel. The KHAW/Workit runtime that creates login links needs access to it as a secret.
