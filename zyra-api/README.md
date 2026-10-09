# Zyra API

Go 1.26 API using Gin, GORM/PostgreSQL, JWT and bcrypt. The API owns report parsing/evaluation, persistence, authorization and ticket lifecycle. It is a development implementation, not a production-ready service.

See the [project overview and Docker demo](../README.md), [product requirements](../documentation.md) and [implemented API contract](doc/API.md).

## Current capabilities

- JWT access tokens lasting up to 15 minutes; rotating refresh sessions with a fixed seven-day deadline from sign-in. Login is required again at that deadline.
- Owner/admin/trusted/normal permissions, active/retired lifecycle and server-side session/account checks. Retirement revokes all sessions while preserving history.
- Profiles, HTTPS avatar URLs, appearance/theme preferences, private bookmarks, active-user mention lookup and recipient-owned notifications.
- Client/database configuration, selected checks, resource thresholds, email sources and expected-delivery windows.
- Manual daily-report submission, immutable assessment settings/results, raw-body access and transactional assessment/ticket/system-findings creation.
- Initial Windows/Linux parsing, including drives/mounts, grouped backup/tablespace findings and incomplete reports. `Backups=NOT_US` never creates a backup issue.
- Lightweight ticket lists, details, paginated timelines, rich comments/direct HTTPS GIF URLs, author-only edits/deletion, Close, Comment and close, Reopen and similar issues.
- Manually triggered missing-email evaluation, not an automatic scheduler or mailbox consumer.

Comment and close requires meaningful content; Close is independent. Comments on closed tickets do not reopen them. Editing/deleting a closure comment does not undo its immutable status event. Similar issues select up to five matches for the same client/database/check type, excluding the current ticket: newest open match first, then recent remaining matches. No merging occurs.

## Code organization

```text
cmd/server/          Startup and dependency wiring
cmd/seed-demo/       Development-only fictional report fixtures
internal/auth/       JWT and token helpers
internal/config/     Process environment configuration
internal/database/   PostgreSQL connection and development migrations
internal/models/     Domain models
internal/repository/ Domain repositories and transaction support
internal/services/   Use cases, permissions, parsing and evaluation
internal/handlers/   HTTP request/response handling
internal/middleware/ Authentication and access checks
internal/routes/     Routes and HTTP integration tests
internal/apperrors/  Shared application errors
doc/API.md           Endpoint contracts and limitations
```

Use domain-specific files in each applicable layer, such as `ticket_model.go`, `ticket_repository.go`, `ticket_service.go` and `ticket_handler.go`. Shared helpers and dependency containers stay shared. Handlers call services rather than repositories directly.

## Local development

Commands below run from `zyra-api/`. Requires Go 1.26 and PostgreSQL; Docker Compose can provide a disposable development/test database:

```sh
docker compose -f compose.test.yaml up -d --wait
go mod download
export APP_ENV=development DEV_ADMIN_LOGIN=true AUTO_MIGRATE=true
export DATABASE_URL='host=127.0.0.1 port=55432 user=zyra_test password=zyra_test_local_only dbname=zyra_test sslmode=disable'
export JWT_SECRET=local-development-secret-not-for-deployment
export ADDRESS=127.0.0.1:8081
go run ./cmd/seed-demo
go run ./cmd/server
```

`seed-demo` is optional and creates three fictional clients/databases and 10 reports producing 30 tickets (21 open, 9 closed). It skips existing reports and preserves operator edits. The fixture login is `admin` / `admin`, with owner role. Never use these credentials or this secret outside local development.

The test database is bound to `127.0.0.1:55432` and uses temporary in-memory storage. Stopping/removing it loses its data; use the [root Docker demo](../README.md#local-demo) when local persistence is wanted.

The API lives under `/api`; `/health` checks HTTP process liveness, not database readiness. With the example above it listens on `127.0.0.1:8081`. Start the frontend separately using the [web instructions](../zyra-web/README.md#local-development).

### Configuration and owner setup

The API reads process environment variables; it **does not automatically load `.env`**. See [.env.example](.env.example). `DATABASE_URL` and a `JWT_SECRET` of at least 32 characters are required. The default listener is `127.0.0.1:8080`.

For a non-demo database, optional `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` create an account only when its email is absent. Passwords must be 12–72 bytes. If no owner exists, set `OWNER_EMAIL` to an explicitly chosen existing active account; bootstrap can create it first. Startup requires exactly one active owner. No arbitrary first-user promotion or existing-owner replacement occurs.

`DEV_ADMIN_LOGIN=true` requires `APP_ENV=development` and a loopback listener. It creates/promotes the known demo fixture without resetting its password; a different existing owner is a conflict. Disabling this flag denies the fixture's login, access and refresh even if its record remains. The root Docker demo additionally uses `DEV_ADMIN_CONTAINER=true` for its private unspecified-IP listener; that flag is not protection against exposing a container publicly.

`AUTO_MIGRATE=true` creates/updates development tables. Production schema migration policy is not established. The current Docker image also runs demo seeding before startup and must not be reused unchanged for production.

## Checks

```sh
go test ./...
go vet ./...
go build ./...
```

PostgreSQL-backed tests are opt-in. Use `ZYRA_TEST_DATABASE_URL` with a **keyword-format DSN** pointing to a disposable test database. Tests create a uniquely named schema and remove it afterward; the database user needs schema creation permission.

```sh
docker compose -f compose.test.yaml up -d --wait
ZYRA_TEST_DATABASE_URL='host=127.0.0.1 port=55432 user=zyra_test password=zyra_test_local_only dbname=zyra_test sslmode=disable' go test -race ./...
docker compose -f compose.test.yaml down
```

Without the variable, database-backed tests are skipped; unit tests still run. The final command removes the disposable container/network and its temporary data, not the root demo's persistent volume.

## API boundaries and remaining work

Lists return ticket summaries; full details, timeline pages and raw email are separate requests. Timeline defaults to 50 events. Filters/sorts are allowlisted, and indexed/batched queries support list and similar-issue paths. Clients/databases have API endpoints, but their frontend pages and proposed identity/mailbox extensions are not implemented yet.

Remaining work includes live mailbox matching/ingestion, background scheduling, recovery-email delivery, the remaining parser catalogue, ticket-note editing, image uploads/GIF search, production migrations/security/deployment and realistic-volume performance verification. Non-OK Recovery Area Space and failed-job rules still need real fixtures; unsupported output remains unknown. Automatic closure/merging and assessment Unresolved/Resolved semantics remain undecided.

SQL monitoring, the separate 30-minute standby-alert flow and authentication rate limiting belong to Release 1.0. External avatars/GIFs are stored as URLs without server media fetching, uploads or a provider resolver.
