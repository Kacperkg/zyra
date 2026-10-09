# Zyra API

Status: initial development implementation. All routes below use the `/api` prefix unless stated otherwise. Responses are JSON except raw-email bodies. Authentication uses `Authorization: Bearer <access_token>`; the frontend persists sessions in local storage, retaining the fixed seven-day deadline and clearing stored credentials on logout/expiry.

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
| PATCH /users/me | Change `name`, `email`, `theme` (light/dark), `appearance` (modern/classic), or `avatar_url`. Cannot change own role/status. |
| POST /users/me/password | Accepts `current_password`, `password`; changes the password and revokes sessions, requiring sign-in again. |

Access expires in 15 minutes, capped at the fixed seven-day session deadline. Refreshing does not extend that deadline. Every authenticated request checks the session and current user, including active/retired status and current role. Retirement revokes existing sessions; reactivation requires a new login. Mutations lock and recheck the actor in the transaction used for the write, so retirement and participation serialize.

Passwords must be 12–72 bytes in this initial implementation. Reset tokens are hashed, single use, and expire after 30 minutes. The reset mechanism is integration-tested with a fake delivery adapter; no email provider has been selected. There is no public registration endpoint.

Local test exception: explicitly setting `DEV_ADMIN_LOGIN=true` with `APP_ENV=development` and a loopback listener bootstraps the isolated `admin@zyra.test` owner fixture with password `admin`. An existing fixture is promoted without resetting its password, and a competing owner is rejected. Login accepts `{"email":"admin","password":"admin"}` in that mode and returns an ordinary token pair. The fixture is denied login, access and refresh when the flag is off. Non-demo startup requires exactly one active owner; when none exists, `OWNER_EMAIL` must explicitly select an existing active account. No arbitrary first-user/admin promotion occurs. See the README for setup.

## Users, clients, and databases

| Method/path | Access and behaviour |
| --- | --- |
| GET /admin/users | Owner/admin; paginated users, with `status=active|retired` filtering. |
| POST /admin/users | Owner/admin; `email`, `name`, `password`, `role` (defaults to normal), status defaults to active. Cannot create another owner. |
| PATCH /admin/users/:id | Owner/admin; profile/role/status subject to the boundaries below. Obsolete `disabled` writes are rejected. |
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

Owner inherits admin capabilities in all endpoints described as admin-only. Only owner can grant/remove admin or retire/reactivate admins. Admin role/status administration targets trusted/normal users, never other admins or themselves; editing one's own profile remains allowed. The owner cannot be retired or demoted. Combined role/status updates cannot bypass these boundaries. Retirement preserves identity, profile, comments, bookmarks and notification history. Role/status transitions append an internal audit record with actor, target, previous/new values and timestamp in the same transaction.

Profile image URLs are at most 2048 bytes, require HTTPS and a hostname, and reject embedded credentials. Empty string clears the image. The API stores the URL without fetching/proxying it. `appearance` defaults to `modern` and is independent of `theme`; preferences are returned on login and profile reads. The frontend profile page now exposes these preferences and private saved tickets.

## Private saved tickets and notifications

| Method/path | Behaviour |
| --- | --- |
| GET /users/me/saved-tickets | Caller-only summaries, `q`, `page`, `limit`; newest saved first, ticket ID breaks ties. |
| PUT /users/me/saved-tickets/:ticket_id | JSON `{ "title": "optional personal title" }`, or `{}` to save without changing an existing title. Unique caller/ticket identity; repeated saves and retitling preserve original saved time. Empty title clears it. |
| DELETE /users/me/saved-tickets/:ticket_id | Idempotent removal of the caller's bookmark only. |
| GET /users/mention-options | Name search `q` requires two characters; shorter queries return empty `items`. `limit` defaults to 20, maximum 20. Active accounts only; minimal `id`, `name`, `avatar_url`, no email or session material. |
| GET /users/me/notifications | Caller-only active notifications; optional `unread=true|false`, `page`, `limit`. Opening the list does not mark anything read. |
| GET /users/me/notifications/unread-count | `{ "count": n }`; excludes withdrawn notifications. |
| PATCH /users/me/notifications/:id/read | Idempotent owned read operation; preserves original read time. Foreign or withdrawn IDs return 404. |
| POST /users/me/notifications/read-all | Marks only caller-owned, nonwithdrawn unread notifications created at or before the request cutoff. Later notifications stay unread. |

Saved-ticket and notification lists use standard `items/total/page/limit`, default 50, maximum 200; page must be 1–1000000. Bookmark titles are trimmed and capped at 200 bytes; bookmark search at 500 bytes; mention search at 200 bytes. Unknown request fields, including forged owner IDs, are rejected. Bookmark summaries include normal compact ticket fields plus `personal_title`, `display_title` (personal title or ticket title), and `saved_at`; neither reports nor discussions are embedded. Ticket detail includes only the caller's bookmark state/title.

Notifications return `id`, `ticket_id`, `ticket_number`, `ticket_title`, `comment_id`, `actor_id`, `actor_name`, `actor_avatar_url`, `created_at`, and nullable `read_at`, newest first with ID tie-breaking. A persistent comment/recipient delivery ledger enforces once-ever notification: removing a mention withdraws the notification; re-adding it does not create or reactivate one. Deleting a comment withdraws read and unread notifications. Notifications have no automatic expiry in this iteration. The frontend bell polls every 30 seconds and stops on sign-out; no live delivery infrastructure is added.

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
| GET /tickets/:id | Ticket, client/database, participants, caller-only `saved_ticket` (or null), and up to five matching similar tickets; timeline events are not embedded. |
| GET /tickets/:id/events | Timeline in pages of 50 events. |
| POST /tickets/:id/comments | Requires meaningful rich content or legacy `comment`; commenting on a closed ticket does not reopen it. |
| PATCH /tickets/:id/comments/:comment_id | Author-only replacement with current `revision`; stale revisions return 409. |
| DELETE /tickets/:id/comments/:comment_id?revision=N | Author-only deletion with current revision; erases text and withdraws notifications. |
| POST /tickets/:id/close | Comment-free closure. |
| POST /tickets/:id/comment-and-close | Requires meaningful rich content or legacy `comment`; comment and separate closure event are atomic. |
| POST /tickets/:id/reopen | Explicit comment-free action; requires a closed ticket. |

Active normal, trusted, admin and owner users can comment/close/reopen. Repeated close/reopen requests return a conflict. History remains independent of assessment results. No role can edit/delete another author's comments or system/status events.

Comments use the versioned contract below, including GIFs by direct HTTPS URL; legacy plain text remains accepted. The frontend editor/renderer supports the text/GIF/mention contract, author edits/deletion and safe links. Image uploads and separate ticket-note editing remain deferred. Database/client notes are supported.

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
- Add image/file uploads only after media storage is agreed; GIFs and avatars currently use direct HTTPS URLs.
- Resolve schedule grace periods, source matching, administrator bounds, and late-arrival policy.
- Validate large-history query performance and production schema constraints/migrations.
- Add production deployment configuration and the agreed production security/operations measures. Authentication rate limiting is explicitly Release 1.0 scope, not PoC scope.

Do not treat this first API pass as completion of every PoC acceptance criterion.

## Rich comment JSON contract (version 1)

New comments and comment-and-close accept:

```json
{
  "schema_version": 1,
  "content": {
    "blocks": [
      {
        "type": "paragraph",
        "align": "left",
        "children": [
          {"type": "text", "text": "Please investigate ", "bold": true},
          {"type": "mention", "user_id": "USER_ID"},
          {"type": "text", "text": " — runbook", "href": "https://example.com/runbook"},
          {"type": "gif", "src": "https://example.com/reaction.gif", "alt": "Investigation complete"}
        ]
      }
    ]
  }
}
```

- `blocks`: 1–100 non-nested blocks. Types: `paragraph`, `bullet_list`, `ordered_list`, `code_block`. Each list block represents one item; adjacent matching list blocks form a list in a future renderer. Alignment is optional `left`, `center`, or `right`.
- Text nodes allow `bold`, `italic`, `underline`, `strike` booleans; optional `size` of 12/14/16/18/20/24; optional six-digit `#RRGGBB` colour; and optional absolute HTTP(S) `href` up to 2,048 bytes, without credentials.
- Mention nodes contain only `type` and `user_id`. The server validates identity and derives the plain-text name. No caller-provided mention list or plain-text projection is accepted.
- GIF nodes contain `type: "gif"`, `src` and optional `alt`. Sources must be absolute HTTPS URLs without credentials, up to 2,048 bytes. Descriptions allow up to 200 bytes; at most 10 GIFs per comment. GIF nodes cannot carry text, mention, link or formatting fields; text/mention nodes cannot carry GIF fields. GIF-only comments and comment-and-close are valid. The server stores URLs without fetching, proxying or checking the response's media type; a `.gif` suffix is not required because direct media endpoints may lack extensions. There is no provider page-link resolver or GIF search integration. Uploads and general image nodes remain excluded. The browser renderer loads the media URL directly and handles unavailable/invalid media.

- Maximum 1,000 inline nodes, 20 unique recipients, 20,000 text bytes and 64 KiB serialized content. At least one non-whitespace text node, valid mention or valid GIF is required. Unknown fields/nodes, raw HTML nodes, arbitrary styles, unsupported versions and image nodes are rejected. Literal text and GIF descriptions are data and must be escaped by renderers, never inserted as HTML. GIF preview text is `[GIF]` or `[GIF: description]`; descriptions and URLs do not create mentions.
- Legacy `{ "comment": "text" }` converts to a paragraph document. Do not combine legacy and rich content. The API does not auto-link text: the later editor can detect typed URLs and emit validated `href` nodes.
- PATCH uses the same body plus `revision`, starting at 1. Creation must omit revision. DELETE uses `?revision=N`. Missing/invalid revision returns 400; stale revision 409; another author 403; unavailable/deleted comment 404. System/status event IDs are not editable comment IDs and return 404.
- Writes retain the existing ticket response. Fetch `/tickets/:id/events` to retrieve comment identity and revision. Comment events include `comment_id`, `schema_version`, `content`, `revision`, `edited_at` when edited, and server-derived `comment`. Events with mentions also include deduplicated `mention_users` identities (`id`, `name`, `avatar_url`), including retired historical recipients; these are read-only response metadata, never accepted as writable labels. The page batches identity lookup rather than making a query per mention. Edits preserve `created_at` and position.
- Deletion erases JSON/text, hides the comment event, retains minimal identity/revision/timestamps and the delivery ledger, and withdraws all notifications. The separate closure event remains visible. Historical participants remain attributed, even after their comment is deleted.
- New mentions require active accounts; edits may preserve an existing mention of a now-retired person without new delivery. Self-mentions do not notify. Removed/re-added recipients are never notified twice for one comment, and withdrawn notifications never reappear. Notifications have no automatic expiry yet.
- New comment creation has no retry/idempotency key: retrying after an ambiguous network failure may create another comment. Revision checks protect edits, not duplicate new submissions.

Development migration preserves legacy comment/event IDs and timestamps. For historical comment-and-close, the original ID becomes the comment and its timeline event; an immutable `-closure` suffixed event preserves the closure. Back up the database before migrating; there is no automatic downgrade after text erasure. The migration is transactional and safe to rerun, but is not a substitute for production migration tooling.
