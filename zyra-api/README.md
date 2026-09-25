# Zyra API

Initial Oracle daily-check proof-of-concept API using Go, Gin, GORM/PostgreSQL, JWT, and bcrypt. This is a development implementation, not the completed PoC or a production-ready release.

See [product requirements](../documentation.md) for agreed behaviour and the [API contract](doc/API.md) for implemented endpoints, limitations, and remaining work. Design requirements are not claims of implemented functionality. SQL monitoring and the separate 30-minute standby-alert flow are Release 1.0 scope.

## Current capabilities

- Sign-in, rotating refresh tokens, logout and session revocation. JWT access lasts up to 15 minutes; the session ends seven days after the original sign-in and requires signing in again. Refresh never extends that deadline.
- Admin/trusted/normal roles, user management, profile details and password-reset logic. Recovery-email delivery is not configured.
- Clients, databases, selected checks, resource thresholds, email sources and schedule configuration.
- Manual daily-report submission, assessment history, raw-body access and transactional ticket creation. This is not live mailbox ingestion.
- Initial Windows/Linux report parsing, including filesystem drive/mount rows, grouped backups, tablespaces and received-but-incomplete reports. Non-OK FRA and failed-job evaluation still need real fixtures and rules; unsupported output remains unknown.
- Lightweight ticket summaries, separate ticket details and 50-event timeline pages, system findings, comments, separate close/comment-and-close actions, reopen, similar issues, administrator closure history and an open Oracle issue count.
- Manually triggered expected-email-window evaluation; no background scheduler yet.

## Code organization

The Go application lives in `zyra-api/`; the React application belongs in `zyra-web/`.

```text
cmd/server/          Startup and dependency wiring
internal/auth/       JWT and token helpers
internal/config/     Environment configuration
internal/database/   PostgreSQL connection and development migration
internal/models/     Domain models
internal/repository/ Typed domain repositories and transaction support
internal/services/   Use cases, permissions, parsing and evaluation
internal/handlers/   HTTP request/response handling
internal/middleware/ Authentication and access checks
internal/routes/     Route registration and HTTP integration tests
internal/apperrors/  Shared application errors
doc/                 API contract and remaining-work documentation
```

Use domain-specific files in each applicable layer, such as `ticket_model.go`, `ticket_repository.go`, `ticket_service.go` and `ticket_handler.go`. Shared helpers remain shared. Services and handlers use shared dependency containers; handlers call services rather than repositories directly.

## Run

Use Go 1.26 and PostgreSQL. For local development you can start the isolated test database included in this directory:

```sh
docker compose -f compose.test.yaml up -d --wait
```

It listens only on `127.0.0.1:55432`, uses non-production credentials, stores its data in a temporary in-memory filesystem, and is intentionally discarded when the container is removed. It is suitable for development and integration tests, not deployment or persistent application data.

Set the variables described in [.env.example](.env.example) in your process environment, then:

```sh
go mod download
go run ./cmd/server
```

The application does not load `.env` automatically. `DATABASE_URL` and a `JWT_SECRET` of at least 32 characters are required. Set both bootstrap-admin variables to create the initial account; choose a password of 12–72 bytes. An existing account is not overwritten.

Set `AUTO_MIGRATE=true` to create the development tables. Production migration policy is not established. The default listener is `127.0.0.1:8080`; `ADDRESS` overrides it. The API lives under `/api`, and `/health` reports that the HTTP process is running.

### Local frontend test account and sample data

For the disposable Compose database, enable the explicitly local `admin/admin` fixture:

```sh
export APP_ENV=development DEV_ADMIN_LOGIN=true AUTO_MIGRATE=true
export DATABASE_URL='host=127.0.0.1 port=55432 user=zyra_test password=zyra_test_local_only dbname=zyra_test sslmode=disable'
export JWT_SECRET=local-development-secret-not-for-deployment
export ADDRESS=127.0.0.1:8081
go run ./cmd/seed-demo
go run ./cmd/server
```

The optional seed command creates three fictional clients/databases and 48 reports producing 144 tickets (108 open, 36 closed), with system findings and example closure comments. Re-running skips existing reports and preserves operator edits. Use it only on disposable development data. Removing the Compose container discards the account and reports; re-run the commands to recreate them.

`DEV_ADMIN_LOGIN` defaults to false and requires `APP_ENV=development` plus a loopback IP listener. It creates a normal **admin** role account at `admin@zyra.test` with username alias `admin` and password `admin`. There is no superadmin role. Existing accounts are never reset by startup. Disabling the flag blocks this fixture's login, access tokens and refresh tokens even if its database record remains. Ordinary user creation still requires a valid email and a 12–72-byte password.

Use `API_PROXY_TARGET=http://127.0.0.1:8081 npm run dev` from `zyra-web/` for this listener. The browser uses the shared login screen for all roles; no public registration is provided. The known password and development JWT secret above must only be used locally.

## Checks

```sh
go test ./...
go vet ./...
go build ./...
```

PostgreSQL integration tests are opt-in. Supply a **keyword-format DSN** using `ZYRA_TEST_DATABASE_URL` for a disposable development/test database. Tests create a uniquely named schema and remove only that schema afterward; the database user needs schema creation permission. With the Compose service running:

```sh
ZYRA_TEST_DATABASE_URL='host=127.0.0.1 port=55432 user=zyra_test password=zyra_test_local_only dbname=zyra_test sslmode=disable' go test -race ./...
docker compose -f compose.test.yaml down
```

Without that variable, the PostgreSQL workflow test is skipped. Unit tests still run. `down` removes the temporary container and network; there is no persistent database volume to delete.

## Implemented ticket API decisions

- Lists return lightweight ticket summaries; ticket details, 50-event timeline pages and raw email are separate requests.
- New tickets begin with a system findings event built from assessment-time findings and thresholds.
- Comment, comment-free Close, required-comment Comment and close, and Reopen are separate actions. Comments on closed tickets do not reopen them.
- Similar issues return up to five tickets for the same client, database and issue type, excluding the current ticket. Pin the newest open match, then fill with recent remaining matches regardless of status. Matches may be older or newer; creation time then ID descending defines recency. Selection uses one database query and never merges or closes tickets.
- Ticket filters support client/database/check/assessment/status/ticket number and created-date boundaries, with allowlisted sorting including client and database names.
- Targeted PostgreSQL indexes support ticket lists, similar issues, event timelines, participants, closure history and assessment schedule/source lookups.

## Agreed frontend and product work still to implement

- Separate Open/Closed issue views, reached through the top navbar or dashboard; no Open/Closed switch on the issues page.
- Fetch 50 ticket summaries initially, another 50 automatically on scroll, then Next page at 100. The UI page and API batch are distinct.
- Fetch ticket detail on navigation and raw email on opening its tab. The ticket interface has Discussion and Raw Email tabs and separately labelled Database notes and Ticket notes; ticket-note storage/editing still needs implementation.
- Retain historical assessment results independently of ticket status. Automatic closure is deferred. Assessment Unresolved/Resolved semantics and repeated-failure grouping remain undecided.
- Keep clients shared across database engines; SQL functionality remains deferred to Release 1.0.

The frontend will use React with Vite, TanStack Router, React Context and colocated CSS Modules. The product documentation defines its folder plan, shared theme tokens, two-column Oracle/SQL dashboard, mutually exclusive animated navbar dropdowns, avatar placement, green accent and neutral charcoal dark theme. A navbar-width dropdown is a mockup candidate. These are frontend requirements; the mockups are not wired to this API.

Other outstanding work includes mailbox matching/ingestion, a background scheduler, recovery-email delivery, the remaining parser catalogue, rich text/media and avatars, production deployment containers, production migrations/security and performance verification with realistic data volumes. Authentication rate limiting is Release 1.0 scope rather than PoC scope. Non-OK Recovery Area Space rules must wait for a real failing example.

## Review workflow

Keep changes local for review. Do not commit or push without explicit approval.
