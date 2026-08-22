# Deploying colabora-be

Production deploy target: a self-managed VPS running Docker + the Compose plugin, with GitHub Actions building/pushing the image to GHCR and deploying over SSH on every push to `main`.

## One-time VPS setup

1. Provision the VPS, install Docker + the Docker Compose plugin.
2. Create a dedicated deploy user (avoid deploying as root). Add its SSH public key as an authorized key on the VPS.
3. Add these secrets in GitHub (`Settings → Secrets and variables → Actions`):
   - `VPS_HOST` — VPS IP or hostname
   - `VPS_USER` — the deploy user
   - `VPS_SSH_KEY` — the matching private key
4. `mkdir -p /opt/colabora-be` on the VPS.
5. Create `/opt/colabora-be/.env` by hand on the VPS (copy from this repo's `.env.example`, fill in real `DB_*`, `JWT_SECRET`, `SMTP_*`, and set `APP_ENV=production`, `NGINX_PORT`). **Never commit this file.**
6. If the `ghcr.io/pln-colabora/colabora-be` package is private (default), the VPS needs pull access: run `docker login ghcr.io -u <github-username>` once on the VPS using a PAT with `read:packages` scope. Simpler alternative: make the package public in GitHub (`Settings → Packages`), which needs no VPS-side login at all.
7. In Cloudflare, point `api-colabora.anargya.fun` (proxied, orange cloud is fine) at the VPS's IP, and set SSL/TLS mode to **Flexible** (`SSL/TLS → Overview`). Flexible means Cloudflare terminates HTTPS for visitors and talks to the VPS over plain HTTP — the origin nginx (`docker/nginx/default.conf`) is already configured for plain HTTP only, with no certbot/TLS on the origin. Do **not** add an HTTP→HTTPS redirect at the origin — with Flexible mode, Cloudflare always connects to origin over HTTP, so an origin-side redirect causes a redirect loop.

## Deploying

Push to `main`. GitHub Actions runs three jobs: `test` → `build-and-push` (builds `docker/Dockerfile.prod`, pushes to `ghcr.io/pln-colabora/colabora-be` tagged `:latest` and `:sha-<short-sha>`) → `deploy` (copies `docker-compose.prod.yml`, `docker/nginx/default.conf`, and `scripts/deploy.sh` to `/opt/colabora-be` on the VPS over SSH, then runs `scripts/deploy.sh`, which pulls the new image, runs migrations, rolls the app container, and verifies `/health`).

No manual steps are needed after the one-time setup above — the first push creates `docker-compose.prod.yml` on the VPS itself.

## Rollback

SSH into the VPS and run:

```bash
cd /opt/colabora-be
IMAGE_TAG=sha-<previous-short-sha> ./scripts/deploy.sh
```

Find `<previous-short-sha>` from a prior successful GitHub Actions run, or `git log --oneline` locally.

## Known limitations (not covered by this setup)

- **Cloudflare↔origin traffic is plain HTTP** (Flexible mode) — only visitor↔Cloudflare is encrypted. If Cloudflare↔origin encryption is needed later, switch Cloudflare's SSL/TLS mode to Full (strict) and issue a free Cloudflare Origin CA certificate for nginx (`SSL/TLS → Origin Server` in the Cloudflare dashboard) — simpler than certbot/Let's Encrypt since it doesn't need port 80 open for ACME challenges or a renewal cron.
- **Migrations run automatically before cutover**, which is safe for additive/`AutoMigrate`-style changes but not for destructive/breaking schema changes — those need manual coordination.
