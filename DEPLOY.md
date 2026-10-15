# Deploying colabora-be

The repository has two deployment targets:

| Target | Branch | Host | Trigger |
| --- | --- | --- | --- |
| Staging | `main` | Homeserver behind Cloudflare Tunnel | Every push to `main` |
| Production | `production` | Existing VPS | Every push to `production` |

`main` is intentionally direct-push friendly for development and staging. The
`production` branch must be protected in GitHub so changes arrive through a pull
request, but the production GitHub Environment must have **no required reviewers**.
Production deployment starts automatically after a successful push to `production`.

The workflow builds immutable GHCR images using the full commit SHA:

```text
ghcr.io/pln-colabora/colabora-be:sha-<full-commit-sha>
```

It also updates the `main` or `production` branch tag. Deploy scripts always receive
the immutable SHA tag, so `latest` is not used for a rollout.

## GitHub configuration

Create two GitHub Environments:

### `staging`

Add these environment secrets:

- `STAGING_SSH_HOST` — Cloudflare Access SSH hostname, for example `staging-ssh.example.com`
- `STAGING_SSH_USER` — dedicated staging deploy user
- `STAGING_SSH_KEY` — private key for that user
- `CF_ACCESS_CLIENT_ID` — Cloudflare Access service-token client ID
- `CF_ACCESS_CLIENT_SECRET` — Cloudflare Access service-token client secret

Do not configure required reviewers or a wait timer for this environment.

### `production`

Add these environment secrets:

- `VPS_HOST` — VPS IP or hostname
- `VPS_USER` — dedicated production deploy user
- `VPS_SSH_KEY` — private key for that user

Do not configure required reviewers for this environment. The CI job references the
environment only to scope production secrets.

## Branch protection

Configure GitHub branch protection/rulesets as follows:

- `main`: direct push allowed; CI runs on pull requests and pushes.
- `production`: direct push disabled; require pull request and successful CI status.
- Disable force-push and branch deletion on `production`.
- Do not add required reviewers to the `production` deployment environment.

The repository workflow cannot create these GitHub settings automatically; they are
repository-level controls and must be configured once in GitHub Settings.

## Production VPS setup

1. Install Docker and the Docker Compose plugin.
2. Create a dedicated deploy user and install the matching SSH public key.
3. Create the deployment directory as root/sudo and give it to the deploy user:

   ```bash
   sudo mkdir -p /opt/colabora-be
   sudo chown "$VPS_USER":"$VPS_USER" /opt/colabora-be
   ```

4. Create `/opt/colabora-be/.env` manually from `.env.example`. Set real production
   values, `APP_ENV=production`, and the production port configuration. Never commit
   this file.
5. Authenticate the host to GHCR using a read-only package credential, or make the
   package public if that is acceptable for this deployment.
6. Keep production PostgreSQL and Garage volumes on this VPS. Do not reuse staging
   credentials or volumes.
7. Keep the existing production Cloudflare DNS mapping. The origin currently uses
   plain HTTP behind Cloudflare Flexible mode; changing to Full (strict) requires an
   origin certificate and a coordinated nginx change.

## Staging homeserver setup

1. Install Docker, Docker Compose, SSH, and `cloudflared`.
2. Create a dedicated deploy user and `/opt/colabora-be-staging`.
3. Create a staging `.env` manually. It must use different values for database,
   JWT, Garage bucket/credentials, and SMTP from production. The Compose defaults are:

   ```dotenv
   APP_NAME=colabora-be-staging
   APP_ENV=staging
   GOLANG_PORT=8888
   NGINX_PORT=8081
   DB_HOST=postgres
   GARAGE_S3_ENDPOINT=http://garage:3900
   GARAGE_BUCKET=colabora-documents-staging
   ```

4. Authenticate the homeserver to GHCR with read-only package access.
5. Create a Cloudflare Tunnel route for the staging API:

   ```text
   staging-api.<your-domain> → http://127.0.0.1:8081
   ```

6. Create a protected Cloudflare Access SSH route:

   ```text
   staging-ssh.<your-domain> → ssh://127.0.0.1:22
   ```

7. Add a Cloudflare Access Service Auth policy for the CI service token. The
   `cloudflared` client version in the workflow is pinned; upgrade it only after
   testing non-interactive SSH from GitHub Actions.
8. On Fedora, keep SELinux enforcing. The staging Compose file applies private `:Z`
   labels to the nginx and Garage configuration bind mounts.

The staging nginx port is bound to loopback only. Public API access occurs through
the Tunnel; no public inbound application port is required on the homeserver.

## Deployment behavior

### Push to `main`

GitHub Actions runs:

```text
test → build/push SHA image → Cloudflare Access SSH → staging docker compose pull → migration → rollout → healthcheck
```

The homeserver performs the actual GHCR pull. GitHub Actions only connects through
Cloudflare Access and invokes the remote deployment script.

### Merge to `production`

GitHub Actions runs:

```text
test → build/push SHA image → VPS SSH → production docker compose pull → migration → rollout → healthcheck
```

There is no reviewer approval pause. The only release gate is the protected
`production` branch and successful CI.

## Rollback

Find a previous successful image SHA in GitHub Actions, then run on the target host:

Production:

```bash
cd /opt/colabora-be
IMAGE_TAG=sha-<previous-full-sha> ./scripts/deploy.sh
```

Staging:

```bash
cd /opt/colabora-be-staging
IMAGE_TAG=sha-<previous-full-sha> \
DEPLOY_DIR=/opt/colabora-be-staging \
COMPOSE_FILE=docker-compose.staging.yml \
HEALTHCHECK_URL=http://127.0.0.1:8888/health \
./scripts/deploy.sh
```

Back up production PostgreSQL and Garage data before migrations that are not purely
additive. The deployment script intentionally does not attempt destructive database
rollback automatically.
