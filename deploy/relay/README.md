# Relay + Authentik deployment

This stack is intentionally separate from the COLABORA application database and
Garage volumes. It is currently an infrastructure pilot: the application still
uses Resend until the API migration is implemented and verified.

## Hostnames

| Environment | Relay | Authentik | Instance domain |
| --- | --- | --- | --- |
| Production | `relay.colabora.my.id` | `auth.colabora.my.id` | `mail-relay.colabora.my.id` |
| Staging | `relay.anargya.fun` | `auth.anargya.fun` | `mail-relay.anargya.fun` |

The application nginx remains the public edge. Relay's web process listens on
host port 8090 and is reached through nginx. Relay's SMTP/DNS listeners retain
their required host ports 25, 53, and 587.

## Before starting

1. Ensure the application stack is already running so its Docker network exists:
   `colabora-be_app-network` in production or `colabora-be-staging_app-network`
   in staging.
2. Copy the matching environment example to `.env`, replace every placeholder,
   and set file mode 600.
3. Replace floating image tags with reviewed, pinned releases.
4. Do not expose ports 8090, 15432, or 15433 publicly. Production Relay also
   requires static IPv4, outbound SMTP port 25, DNS port 53 TCP/UDP, PTR/rDNS,
   and the Relay-generated NS delegation.
5. Create the Cloudflare/Tunnel routes for the three hostnames before testing
   OIDC. Staging remains Tunnel-only and is not a deliverability test.

## Start and verify

```sh
docker compose -f compose.yaml config
docker compose -f compose.yaml up -d
docker compose -f compose.yaml ps
docker compose -f compose.yaml logs --tail=100 relay authentik-server authentik-worker
docker compose -f compose.yaml exec relay bin/console verify
```

Open Authentik first, create the `relay-admin` user with TOTP, then create an
OIDC provider/application for Relay. Configure the Relay callback URL as:

```text
https://relay.<environment-domain>/api/oidc/callback
```

Run the resource guard from the host with a systemd timer. It stops only this
stack when available RAM falls below 256 MiB; it does not stop the COLABORA app,
PostgreSQL, nginx, or Garage.
