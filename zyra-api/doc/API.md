# Zyra API

Status: initial development implementation. All routes below use the `/api` prefix unless stated otherwise. Responses are JSON except raw-email bodies. Authentication uses `Authorization: Bearer <access_token>`; browser storage remains a frontend decision.

The approved stack for this implementation is Gin, GORM/PostgreSQL, JWT, and bcrypt. Current choices are HS256 access JWTs, bcrypt passwords, hashed random refresh tokens rotated on use, and explicit opt-in development `AutoMigrate`. These choices are reviewable. The production deployment and migration policies remain open.

## Code organization

Each domain has its own applicable model, repository, service, and handler files under `internal/`, named `<domain>_<layer>.go`. Tickets, clients, databases, users, assessments, check settings, and email sources follow this convention. Authentication uses separate session/reset-token models and repositories. Schedules are persisted within email sources; dashboard counts use the ticket repository.

Domain repositories have typed interfaces and constructors. Private shared persistence helpers avoid duplicating GORM mechanics; the shared `Store` coordinates transactions across repositories. `Service` and `Handler` remain shared dependency containers, with their methods split into domain files. Validation, pagination, and response helpers stay shared where appropriate. This is a code-organization change, not a change to endpoints or persistence schema.

## Authentication

| Method/path | Behaviour |
| --- | --- |
| POST /auth/login | Accepts `email`, `password`; returns access/refresh tokens, both expiry timestamps, and the user. |
| POST /auth/refresh | Accepts `refresh_token`; rotates it and issues new access within the original session deadline. |
| POST /auth/logout | Authenticated; revokes the current session and its access immediately. |
| POST /auth/forgot-password | Accepts `email`; uses an injected recovery delivery adapter. Returns 503 in the current server because delivery is not configured. |
| POST /auth/reset-password | Accepts `token`, `password`; consumes a valid single-use reset token and revokes sessions. |
| GET /users/me | Current profile. |
| PATCH /users/me | Change `name`, `email`, or `theme` (light/dark). Role changes remain admin-only; users cannot alter their own role or disable themselves. |
| POST /users/me/password | Accepts `current_password`, `password`; changes the password and revokes sessions, requiring sign-in again. |

Access expires in 15 minutes, capped at the fixed seven-day session deadline. Refreshing does not extend that deadline. Every authenticated request checks the session and current user, including disabled status and current role.

Passwords must be 12–72 bytes in this initial implementation. Reset tokens are hashed, single use, and expire after 30 minutes. The reset mechanism is integration-tested with a fake delivery adapter; no email provider has been selected. There is no public registration endpoint.

Local test exception: explicitly setting `DEV_ADMIN_LOGIN=true` with `APP_ENV=development` and a loopback listener bootstraps the isolated `admin@zyra.test` fixture with password `admin`. Login accepts `{"email":"admin","password":"admin"}` in that mode and returns the ordinary admin session/token response. Normal password/email validation remains unchanged. The fixture is denied login, access and refresh when the flag is off, including persisted sessions. See the README for setup and optional fictional seed data. Superadmin is not part of this PoC.

## Users, clients, and databases

| Method/path | Access and behaviour |
| --- | --- |
| GET /admin/users | Admin; paginated users. |
| POST /admin/users | Admin; `email`, `name`, `password`, `role` (defaults to normal). |
| PATCH /admin/users/:id | Admin; name/email/theme/role/disabled. Disabling revokes sessions. |
| GET /admin/users/:id/closed-tickets | Admin; tickets this user has closed, including tickets later reopened, using retained closure events. |
| GET /clients | Authenticated; paginated clients. |
| POST /clients | Trusted/admin; `name`, optional `notes`. |
| GET /clients/:id | Authenticated detail. |
| PATCH /clients/:id | Trusted/admin; partial `name` and `notes`. |
| DELETE /clients/:id | Admin; archives, preserving history. Active child databases must be archived first. |
| GET /databases | Authenticated; paginated databases. |
| POST /databases | Trusted/admin; `client_id`, `name`, optional `hostname`, `ip`, `notes`. |
| GET /databases/:id | Authenticated detail. |
| PATCH /databases/:id | Trusted/admin; partial details. Client reassignment is rejected to preserve ticket context. |
| DELETE /databases/:id | Admin; archives rather than removing history. |

Responses omit password hashes. Archive status defaults to false when listing; use `archived=true` to list archived records.

## Check and email configuration

- `GET /databases/:id/check-settings`: authenticated.
- `PUT /databases/:id/check-settings`: trusted/admin, replaces the settings.
- `GET /databases/:id/email-sources`: authenticated.
- `POST /databases/:id/email-sources`: trusted/admin, creates a source.
- `GET /email-sources/:id`: authenticated.
- `PUT /email-sources/:id`: trusted/admin, replaces source configuration.
- `DELETE /email-sources/:id`: admin, disables the source.
- `GET /email-sources/:id/schedules`: authenticated.
- `PUT /email-sources/:id/schedules`: trusted/admin, accepts `{"schedules": [...]}`.

Example check settings:

```json
{
  "selected": {
    "backups": true,
    "tablespace": true,
    "missing_email": true
  },
  "resources": {
    "backups": {
      "E:\\ORADATA\\IFSPRD\\APEX01.DBF": { "ignore": false }
    },
    "tablespace": {
      "UNDOTBS1": { "ignore": false, "min_free_percent": 5 }
    }
  }
}
```

Resource names match report values exactly. Filesystem uses `max_used_percent`; Tablespace/ASM use `min_free_percent`. Thresholds are validated in the range 0–100. Unconfigured resources are not evaluated. Configure backup paths explicitly for the first parser version.

An email source contains `sender`, `subject`, `timezone`, `enabled`, and `schedules`. Each schedule has a unique `name`, `start`, and `end` in HH:MM form. Overnight windows are supported. Trusted users can edit schedules in this development implementation; administrator-defined bounds are not implemented yet.

## Assessment pipeline

| Method/path | Behaviour |
| --- | --- |
| POST /databases/:id/assessments | Admin-only manual report submission; accepts `message_id`, `body`, optional `subject`, `sender`, and RFC3339 `received_at`. |
| GET /databases/:id/assessments | Paginated history. |
| GET /assessments/:id | Metadata, results, and settings snapshot. |
| GET /assessments/:id/raw-email | Original body as plain text with HTML execution disabled. |
| POST /admin/schedules/evaluate | Admin; accepts `date` as YYYY-MM-DD, evaluates closed windows, returns number of created issues. |

Manual submission lets the report-to-ticket pipeline run before mailbox access is chosen. It is not an automatic sender/subject matcher. The same message ID and body resubmitted to the same database return the existing assessment; conflicting reuse is rejected.

The initial parser recognises plain-text daily-check structure, extracts report time and execution hostname, and handles:

- Configured RMAN backup rows grouped into one Backups issue.
- `NOT_US` as not managed, with no backup issue.
- Tablespace and ASM free percentages, and filesystem usage percentages.
- Invalid Archive Destinations as a grouped issue with raw evidence.
- `OK!` for selected checks.
- Incomplete output as Missing Email when that check is selected.
- Unsupported non-OK output as `unknown`, never silently passed.

The initial completeness profile requires ReportOn, Database, Database checks, selected sections, and a script footer with a usable `Run by` hostname. A Windows-style `Script Info` heading alone is incomplete. Linux reports may omit that heading when they contain `Script : ...` metadata and `Run by`. Footer text alone never establishes completeness when database-check output is missing.

Filesystem parsing supports Windows `Filesystem Usage` drive rows and Linux `Filesystem Mounted Use%` tables. Configure Windows resources by drive (for example `C:`) and Linux resources by mount point (for example `/mnt/backups`); the full Linux row retains the device/source as evidence. Repeated device names such as `tmpfs` remain distinct by mount. Usage must be strictly above the configured maximum to fail: at a 90% limit, 90% passes and 92% fails. Unconfigured or ignored mounts do not create issues. No implicit threshold is assigned to unconfigured resources.

More script versions, HTML/MIME handling, and fuller rules for FRA, failed jobs, clusterware, indexes, and other catalogue checks need additional fixtures. Failed-job failure lists currently return `unknown`, not passed or a ticket. Numeric FRA evaluation is not implemented. Report time remains the original text; received time is stored as a timestamp.

TODO — Recovery Area Space: `RecoveryAreaSpace=OK!` passes when the check is selected. Non-OK output is expected to indicate a potential issue, but the failure rule and system-comment evidence will be fleshed out once a real failing example is supplied. Until then, non-OK FRA output remains `unknown` and does not create a ticket; excluded FRA checks remain unevaluated.

Assessment/results/tickets are saved in one transaction. The initial repeated-failure policy is one set of issues per distinct report; no cross-report merging or automatic recovery closure is implemented. This policy remains subject to review.

Schedule evaluation uses exact sender/subject and received time in source-local windows, with no grace period yet. It does not create an absence issue when a report (including an incomplete report) already arrived within that window. Re-evaluating a window does not create another issue. Evaluation is manually triggered, not a running background scheduler. Late arrivals remain in history without automatically closing an earlier missing ticket.

## Tickets and dashboard

| Method/path | Behaviour |
| --- | --- |
| GET /dashboard | Open Oracle issue count and `sql.available=false`. |
| GET /tickets | Paginated lightweight summaries with client/database names; no evidence, events, or raw email. |
| GET /tickets/:id | Ticket, client/database, participants, and up to five matching similar tickets with the newest open match pinned first; timeline events are not embedded. |
| GET /tickets/:id/events | Timeline in pages of 50 events. |
| POST /tickets/:id/comments | Requires non-empty `comment`; commenting on a closed ticket does not reopen it. |
| POST /tickets/:id/close | Comment-free closure. |
| POST /tickets/:id/comment-and-close | Requires a non-empty `comment`; comment and closure are atomic. |
| POST /tickets/:id/reopen | Explicit comment-free action; requires a closed ticket. |

Authenticated normal, trusted, and admin users can comment/close/reopen in this first version, following the product permission matrix. Repeated close/reopen requests return a conflict. History remains independent of assessment results.

Comments are plain text with a 20,000-byte limit. Rich text, image/GIF uploads, avatars, and separate ticket-note editing are deferred until their representation/storage are chosen. Database/client notes are supported.

The agreed ticket design labels Database notes and Ticket notes separately. Their editing/storage contract and the new UI remain deferred. Automatic closure is not enabled.

## Lists and errors

Agreed UI follow-up: separate Open/Closed issue views, 50 summary rows loaded initially and 50 more automatically on scroll, then explicit Next page after 100 rows. The API returns lightweight ticket summaries, loads detail and 50-event timeline pages separately, and exposes raw email separately for lazy loading. New tickets start with one system findings event. See the product documentation for the confirmed navbar, dashboard, ticket tabs and neutral dark-theme design. Assessment Unresolved/Resolved semantics remain pending.

**Implemented similar-issues selection:** match the same client/database/check type across older and newer tickets, exclude the current ticket, and pin the most recently created open match first. Fill the remaining slots up to five with the most recent matches regardless of status, without duplicating the pinned ticket. With no open match, return the five most recent matches; fewer matches produce fewer results. Sort recency by creation timestamp descending, then ID descending. Return ticket identifiers, numbers, titles, creation timestamps and status for links. A single database statement selects the open candidate independently of the recent candidates, deduplicates them and applies the final order/limit. An older open match is therefore included even outside the latest five. This does not merge tickets or automatically close them.

Lists return `items`, `total`, `page`, and `limit`. Page defaults to 1, limit to 50; maximum limit is 200. Sorting is allowlisted and includes an ID tie-breaker. Use `sort`, `order=asc|desc`, and `q` where supported. Search uses case-insensitive PostgreSQL ILIKE matching; query performance/index tuning remains follow-up work.

Tickets support `client_id`, `database_id`, `check_type`, `assessment_type`, `status`, `closed_by`, and `number` filters, plus `created_from`/`created_to` date or RFC3339 boundaries. Sorting includes ticket number, check type, created time, client name, database name, assessment type and status. Assessment history supports `result` and `type`.

Errors return `{"error": "..."}` with 400, 401, 403, 404, 409, 500, or 503 as appropriate. Unexpected database/internal messages are logged, not returned to clients. Requests are limited to 2 MiB. No browser CORS policy is configured; use a same-origin development proxy until origins are agreed.

## Local integration database

`compose.test.yaml` provides an isolated PostgreSQL service on `127.0.0.1:55432`. It uses non-production credentials, a healthcheck, and temporary `tmpfs` storage. It is test infrastructure only and does not define persistent self-hosted deployment. The exact start, test DSN, and cleanup commands are in the API README.

## Remaining work before a pilot

- Choose and wire mailbox and password-recovery delivery adapters.
- Validate additional real report formats and complete the remaining selected-check rules.
- Add media storage/rich-text handling and profile avatars.
- Resolve schedule grace periods, source matching, administrator bounds, and late-arrival policy.
- Validate large-history query performance and production schema constraints/migrations.
- Add production deployment configuration and the agreed production security/operations measures. Authentication rate limiting is explicitly Release 1.0 scope, not PoC scope.

Do not treat this first API pass as completion of every PoC acceptance criterion.
