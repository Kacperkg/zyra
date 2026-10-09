# Zyra

Zyra is a self-hosted database health-check application. It turns Oracle report emails into assessments and actionable issue tickets, with a Go API and React frontend in one repository.

This is an in-progress proof of concept, not a production-ready release. Reports can currently be submitted manually through the API; live mailbox ingestion is not implemented.

## How it works

1. A report is parsed into one assessment containing the configured check results.
2. Failed checks create separate tickets, grouped by check type: several failing tablespaces produce one Tablespace ticket, for example.
3. Operators investigate findings, comment, close or reopen tickets. Passing assessments create no issue tickets, and closing tickets never changes the original assessment result.

Assessments preserve processing-time settings and evidence. Automatic ticket merging and closure are not implemented.

## Current scope

- Oracle daily-check parsing for Windows/Linux reports, assessment history and manually evaluated missing-email windows.
- Login, owner/admin/trusted/normal roles, active/retired accounts and revoked sessions.
- Dashboard, separate open/closed Oracle lists, ticket discussion/raw email and similar issues.
- Profiles, light/dark themes, Modern/Classic lists, private saved tickets, rich comments, direct GIF URLs, mentions and notifications.
- API configuration for clients, databases, checks, thresholds and email sources.

Client/database/configuration pages are planned, not implemented. Live mail ingestion, background scheduling, recovery-email delivery and some parser rules remain unfinished. SQL monitoring, the separate 30-minute standby-alert flow and authentication rate limiting are Release 1.0 scope.

## Local demo

Requires Docker with Compose. From the repository root:

```sh
docker compose up --build -d
```

Open [localhost:3000](http://localhost:3000) and sign in with **admin / admin**. This development-only account has the owner role. The demo seeds three fictional clients/databases and 10 reports producing 30 tickets on a fresh database; existing operator changes are preserved on subsequent starts.

The web container serves the built frontend through nginx and proxies `/api` to the private API container. Only port 3000 is published, bound to host loopback. PostgreSQL data persists in the `zyra-db` volume.

```sh
docker compose logs -f
docker compose down
```

Stopping the stack preserves the database. **`docker compose down --volumes` deletes the local demo database and all its saved data.**

The Compose stack and API image are for local development: they use known credentials, a development JWT secret, automatic schema migration and demo seeding. Do not deploy them unchanged or expose the demo account publicly. Production deployment and migration policy remain to be designed.

## Repository

```text
zyra-api/          Go API, parsing, persistence and authorization
zyra-web/          React frontend, shared UI and domain pages
compose.yaml       Local full-stack demo
documentation.md   Product requirements and agreed behaviour
```

For separate-process development, use Go 1.26, Node 22.13+ and PostgreSQL. Follow the [API setup](zyra-api/README.md#local-development) and [web setup](zyra-web/README.md#local-development); the API guide includes a disposable test database and local account fixture.

## Checks

Run from the repository root:

```sh
cd zyra-api
go test ./...
go vet ./...
go build ./...
cd ../zyra-web
npm ci
npm test
npm run build
```

Database integration tests require a disposable PostgreSQL database and `ZYRA_TEST_DATABASE_URL`; without it, database-backed tests are skipped. See the [API test instructions](zyra-api/README.md#checks). Frontend browser checks also need a running API.

## Documentation

- [Product requirements](documentation.md): agreed behaviour and remaining decisions.
- [API README](zyra-api/README.md): API setup, architecture, tests and limitations.
- [API contract](zyra-api/doc/API.md): implemented endpoints and payloads.
- [Web README](zyra-web/README.md): frontend setup, structure and interactions.

Requirements and mockups are not claims that a feature is implemented. Keep changes available for review; commit and push only with explicit approval.
