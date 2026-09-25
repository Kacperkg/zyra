# RFC: Completing the Zyra Oracle daily-check proof of concept

Status: Draft for review  
Date: 2026-09-24  
Scope: PoC delivery from the current development API to an internal pilot

## 1. Summary

Zyra turns Oracle health-check emails into durable assessments and actionable tickets. This RFC proposes how to complete the daily-check PoC around the existing Go API, while preserving the product decisions in [documentation.md](documentation.md) and the implemented contract in [API.md](zyra-api/doc/API.md).

This is a new RFC, not a restoration of the previous document. It does not authorize implementation or select infrastructure that remains undecided. **Confirmed** means an existing product decision; **Implemented** means present in the repository; **Proposed** means a recommendation for review; **Open** means a decision is still required. The API is an initial development implementation, not a completed PoC or production release.

## 2. Context and problem

Existing server-side scripts already email Oracle health reports. Operators need a consistent way to distinguish configured failures, healthy runs, incomplete scripts, and absent reports without losing the original evidence. Reports vary across Windows/Linux and script versions; treating any non-empty section as a failure would generate incorrect tickets. Repeated fetches, configuration changes, and late reports also risk duplicate or misleading history.

The implemented API validates much of the report-to-ticket workflow through manual report submission. The initial React UI connects login, dashboard, Oracle lists and ticket detail to that API. Live mail, background schedule evaluation, the remaining product screens and persistent deployment are still incomplete. The next delivery work must connect these boundaries without presenting unresolved product rules as settled.

## 3. Goals and non-goals

**Confirmed PoC goals:** ingest Oracle daily checks from a dedicated mailbox; retain their raw source and results; evaluate only selected/configured checks; detect incomplete and absent reports; support investigation, comments, closure and reopening; enforce normal/trusted/admin roles; and provide a responsive React interface with light/dark themes in a self-hosted Dockerized application.

**Confirmed Release 1.0 work:** SQL monitoring and issue pages, the separate standby-alert email sent every 30 minutes, and authentication rate limiting. These are not PoC implementation tasks. SQL navigation remains a visibly unavailable placeholder during the PoC.

**Non-goals:** executing commands on monitored servers, automatic remediation, SaaS tenancy/billing, native mobile apps, and filling out the placeholder Settings & maintenance navbar menu during the PoC. Database, check, email-source and schedule configuration remain PoC requirements through their defined screens. Automatic ticket closure is deferred; successful or late reports do not close tickets now.

## 4. Current implementation and gaps

| Area | Implemented baseline | Remaining work or decision |
| --- | --- | --- |
| Application | Go/Gin, GORM/PostgreSQL development API; React/Vite frontend for login, dashboard and ticket workflow | Remaining frontend screens and persistent self-hosted deployment |
| Accounts | JWT/bcrypt authentication, rotating refresh sessions, roles, profile fields, session revocation, password reset logic | Browser token storage; recovery delivery adapter; avatars |
| Configuration | Clients/databases, archive behavior, selected checks/resource thresholds, email sources with timezone and windows | Versioned source/parser metadata, grace periods, administrator schedule bounds, richer server records |
| Ingestion | Admin manual submission to a selected database; original body, sender/subject and received time stored | Mailbox adapter, automatic matching, original MIME/source retention contract, retries/review queue |
| Parsing | Initial Windows/Linux plain-text grammar; backups, tablespace/ASM, filesystem, grouped archive failures, completeness checks | Additional real fixtures, remaining check rules, MIME/HTML handling, parser version persistence |
| History | Assessment settings/results snapshots; transactional assessment/ticket/system-event writes | Content-hash idempotency, normalized report timestamps, canonical server identity and explicit unmatched-host warnings |
| Scheduling | Admin-triggered closed-window evaluation; source timezone, overnight windows, repeat-evaluation protection | Background execution, grace, DST policy, late-arrival linking and malformed-window reconciliation |
| Tickets | API and initial UI for summary lists, detail, five similar issues, 50-event pages and comment/close/reopen; API closure reporting | Separate ticket notes contract, controlled rich text/media and remaining administration screens |
| Operations | Opt-in development AutoMigrate and disposable PostgreSQL test Compose service | Production migrations, storage, backups, restore validation, monitoring and performance targets |

Current assessment idempotency uses a globally unique `message_id`: the same ID, database and body returns the existing assessment; conflicting reuse is rejected. Different IDs with identical content are not deduplicated. No provider-scoped identity or deterministic content hash is stored yet.

Report time is currently retained as text, separately from UTC received time. Hostname/IP snapshots exist, but IP is attached only when the parsed hostname matches the database's single configured hostname; the conceptual multi-server/alias model is not implemented. Current source matching for schedule evaluation uses exact sender/subject. Manual submission is not a mailbox matching implementation.

Non-OK FRA and failed-job output remains `unknown`; it is not silently passed and does not generate a failure ticket. The full check catalogue is not implemented. Existing comments are plain text, not rich text.

## 5. Proposed architecture and workflows

### 5.1 Component boundaries

**Confirmed:** retain `zyra-api/` for Go and `zyra-web/` for React. The approved implementation uses Gin and GORM/PostgreSQL; the production database setup and migration policy remain open. Preserve the API's domain-specific layers and shared transaction support.

**Confirmed frontend choices:** Vite, TanStack Router, React Context and colocated `.module.css` files. Use focused authentication/theme contexts and local component state; Redux and Tailwind are excluded. Keep thin route definitions separate from pages, shared UI and domain components, and domain API modules. The initial implementation follows the [frontend organization](documentation.md#113-frontend-organization). Shared server-data caching and persistent browser token storage remain open; the initial implementation uses explicit requests and in-memory tokens.

**Proposed:** introduce replaceable mailbox and recovery-delivery adapters, a durable ingestion lifecycle, a versioned parser contract, and background schedule execution around the existing domain services. A logical worker boundary does not require a separate deployment, queue product, or cache. Choose those only after mailbox capabilities and operational requirements are known.

### 5.2 Ingestion and assessment processing

1. Forward operational reports into the dedicated Zyra mailbox.
2. Persist immutable raw source plus transport identifiers before parsing. Proposed processing states are received, processing, processed, retryable failure, and review required; exact storage and claim/retry mechanics remain open.
3. Match normalized sender/subject to one enabled source. Normalize harmless whitespace/forwarding prefixes only. Retain unmatched or ambiguous messages for administrator review; never guess the database or discard the message.
4. Select the source's parser version; extract metadata and facts, then evaluate against an effective settings snapshot. Verify report database identity where present.
5. Commit the assessment, results, resulting tickets and initial findings events atomically. Retries must not duplicate any of them.
6. Expose retained failures and retry/review actions to operators. A crash between raw persistence and processing must leave recoverable work.

**Proposed idempotency work:** persist provider-scoped message identity and a deterministic content hash with transactional uniqueness. Specify canonicalization and the scope of hash deduplication before implementation: repeated forwarding must be safe without collapsing distinct legitimate runs with similar content. Test concurrent consumers and partial failures. The current manual endpoint remains a diagnostic fixture path, with any compatibility changes documented.

### 5.3 Parsing and ticket invariants

**Confirmed:** preserve raw evidence; separate extracted facts from evaluation; snapshot effective thresholds; and evaluate selected/configured resources only. No default threshold may silently turn an unconfigured resource into an issue. `Backups=NOT_US` never creates a backup ticket. Unknown unsupported output must remain visible as unknown.

Failures within one assessment are grouped by check type into one ticket and one initial system findings event, retaining every affected resource. The example with configured Backups and Tablespace produces exactly two tickets; excluded Indexes/Filesystem produce none. Incomplete reports produce one Missing Email issue when selected and must not produce ordinary issues from partial data. Archive destinations are grouped, retaining the available error evidence without inventing fixed-width column mappings.

**Proposed parser work:** version completeness profiles and fixtures, preserve Windows drives/Linux mount identities, convert ReportOn with the source timezone to a UTC timestamp while retaining original text, and snapshot observed hostname plus matched canonical server/IP identity. Unmatched hosts remain processable with an explicit warning. No ticket view should depend on live DNS.

Cross-report merging is **Open**. The current implementation creates a new set of tickets for each distinct report. Preserve that baseline until a reviewed decision defines occurrence identity, resource grouping, and behavior after closure; do not infer merging from the similar-issues feature.

### 5.4 Missing-email windows and late arrivals

**Confirmed:** distinguish Not received from Received but incomplete, use source-local schedule windows, and create at most one absence issue for a window. A late valid report must not automatically close an issue.

**Implemented nuance:** any exact sender/subject report in a window, including incomplete output, suppresses an absence issue. This avoids creating a second issue for a received-but-incomplete run. Product wording also says a valid assessment satisfies a window. These statements need one explicit reconciliation rule.

**Proposed:** retain distinct window outcomes for valid receipt, malformed receipt, and absence, linking the corresponding assessment/issue. A malformed receipt should not appear healthy, but should not add a redundant absence ticket. Link later valid arrivals to the existing issue and record that the email arrived while leaving ticket status open. This requires a defined state/API contract; it is not implemented and does not settle the separate Unresolved/Resolved assessment semantics.

Before enabling background evaluation, decide window boundary inclusivity, overlaps, grace periods, DST ambiguous/nonexistent local times, source edits after a window, and concurrent receipt/evaluation behavior. Persist a stable window identity so renaming or editing a source cannot silently create duplicate historical issues.

### 5.5 Operator interface

**Confirmed:** use a shared top navbar without a sidebar; separate Open/Closed Oracle issue views; a two-column Oracle/SQL dashboard with SQL disabled; and Discussion/Raw Email ticket tabs with a shared context panel. Follow [product screen requirements](documentation.md#9-screens-and-navigation) for detailed layout and themes.

Only one navbar dropdown is open at a time. Navigation, outside clicks and Escape close it; provide keyboard/focus handling and subtle fade/vertical motion with reduced-motion support. A navbar-width shared panel remains a candidate for mockup review. Use the documented global CSS tokens for the approved light/neutral-dark palette, radii, fonts, shadows and motion; each component/page imports its own CSS Module.

**Confirmed and implemented in the API:** Similar issues match the same client, database and check type, excluding the current ticket, and may be newer or older. Pin the most recently created open match first, then fill up to five results with the most recent remaining matches regardless of status. If no open match exists, use the five most recent. Break equal creation timestamps by ID descending. One database statement selects the open candidate independently of recent matches, deduplicates candidates and applies the final limit. The frontend will show linked ticket number, date/time and status. This helps operators compare a new issue with an unresolved morning report; it does not establish identical findings or introduce manual/automatic merging or closure.

Lists load 50 summaries, automatically load the next 50 on scroll, then offer Next page at 100. Timeline events follow the same 50/100 interaction. Detail and raw source load separately; no bulk prefetch of raw email. Preserve filters in URLs and stable sorting across fetches. Database notes and Ticket notes remain separate; permissions/storage for ticket-note editing require a decision. Assessment Unresolved/Resolved labels must not be implemented with invented semantics.

## 6. API and data considerations

The [API contract](zyra-api/doc/API.md) remains the authority for current `/api` endpoints and payloads. Extend it alongside implementation rather than treating this RFC's logical boundaries as newly available routes.

**Proposed data extensions:** ingestion identity/hash/state and retry metadata; source/parser type and version; raw-source references; normalized report time; canonical server snapshots; stable schedule-window records; and late-arrival links. Occurrence tables are conditional on the cross-report policy. Exact schema, constraints, storage technology and migration format are open. Existing JSONB results/settings/schedules are implementation facts, not a requirement that every future entity remain embedded.

Preserve append-only assessment results and immutable status events. Archive configuration without deleting assessment/ticket history; do not cascade-delete history when a user is disabled. Closing a ticket never rewrites the assessment's historical result. Current APIs expose more outcomes than passed/failed, including unknown, incomplete and not_evaluated; UI filters must account for these without relabeling them as healthy.

**Proposed compatibility rule:** use additive fields/endpoints where possible, document changed submission/deduplication behavior, and migrate with explicit backfill rules. Historical parser versions or timestamps that cannot be recovered reliably must be marked unavailable rather than fabricated. Benchmark existing page/limit queries before selecting a different pagination or caching strategy.

## 7. Security and authentication

**Confirmed and implemented:** HS256 JWT access tokens expire after at most 15 minutes and never exceed the fixed seven-day session deadline. Random refresh tokens are hashed and rotated without extending that deadline. Authenticated requests check current user/session state; logout, disabling and password changes/reset revoke applicable sessions. Passwords use bcrypt; the current length contract is 12–72 bytes. Recovery tokens are hashed, single use and expire after 30 minutes; live delivery is not wired and forgot-password currently returns 503.

The API enforces the confirmed normal/trusted/admin permission matrix. All three roles can comment, close and reopen; trusted/admin users edit configuration; admin users manage roles and archive records. Any future restriction would be a separate explicit product change.

**Confirmed content boundary:** raw mail and external warning banners are untrusted data, never instructions or executable HTML. Current raw bodies are served as text. Rich text requires server-side sanitization and uploads require authenticated access and type/size controls before enabling them.

**Initial frontend session:** access and refresh tokens are held in memory, so a full reload requires sign-in. Concurrent refresh requests are coordinated, and stale responses must not restore a session after logout or account changes. All roles share one login screen; no public registration or superadmin role is included. An explicitly enabled local-development `admin/admin` fixture uses the ordinary admin role and is unavailable when its flag is off. Normal password validation remains unchanged.

**Open before pilot:** persistent browser token storage and associated browser security controls, recovery-email delivery, deployment transport/secrets handling, raw-mail/media retention and access policy, logging/redaction, backups and restore procedures. No cookie/localStorage strategy, storage service, queue or reverse proxy is selected here. Current 2 MiB request limits and absent browser CORS configuration are API facts; they are not a completed deployment security policy. Authentication rate limiting remains committed Release 1.0 work.

## 8. Alternatives and tradeoffs

| Decision | Alternatives | Review guidance |
| --- | --- | --- |
| Mail ingestion | Provider API/events or mailbox polling | Choose from actual provider access, delivery guarantees and operational burden; neither is selected. |
| Background execution | In-process runner or separate worker | Start from recoverability/concurrency requirements; a separate service increases operations but may isolate failures. |
| Repeated failures | New tickets per report or append occurrences to an open ticket | New tickets preserve the current model but may be noisy; merging needs precise resource and recovery semantics. |
| Raw/media storage | Database-backed or external/file storage | Compare size, access enforcement, retention and backup consistency before choosing. |
| Database pagination | Existing page/limit or cursor-based API | Existing UI contract maps to batches; changing strategy requires evidence from realistic volume and concurrent inserts. |

**Proposed sequencing choice:** complete one fixture-backed, end-to-end daily-check slice before broadening parser coverage. Keep unsupported results explicit. This provides pilot evidence without claiming every catalogue rule is complete; mandatory pilot checks must be agreed first.

## 9. Risks and mitigations

- Incorrect matching or unsupported formats can attach failures to the wrong database: require unambiguous matching, identity checks, versioned fixtures and a retained review queue.
- Retries and concurrent workers can duplicate issues: define durable identity/uniqueness and test crash/retry boundaries in PostgreSQL.
- Timezone, source edits and late arrivals can misstate absence: define stable window semantics and test DST/races before background execution.
- Evolving thresholds/server configuration can distort old evidence: retain processing-time snapshots and avoid recalculating history on read.
- Raw email/media can contain sensitive data: resolve access/retention and safe rendering before live pilot data arrives.
- Growing history can make lists slow: validate summary/detail separation and indexed query paths with representative data and agreed performance budgets.

## 10. Proposed rollout and validation

1. **Decision and fixture gate:** choose the pilot mailbox/access method, mandatory checks, anonymized formats, matching rules and schedule semantics; resolve the decisions that block those paths.
2. **Ingestion slice:** implement durable receipt, versioned processing, idempotency and retry/review handling; prove one actual mailbox report becomes exactly one assessment with correct findings.
3. **Schedule slice:** add background execution after agreeing window/grace/DST and malformed/late-arrival reconciliation. Validate concurrency and restart behavior.
4. **Operator slice:** build the agreed React flow, wire recovery delivery, and finish notes/media/profile behavior required by PoC acceptance. Resolve storage and permission decisions before those features.
5. **Pilot gate:** provide persistent Docker deployment, migration/backup/restore procedures, access/retention policy, realistic performance measurements and core end-to-end evidence. Review remaining unknown parser coverage explicitly.

Use the existing [API checks](zyra-api/README.md#checks) as the baseline: Go tests, vet, build and race-enabled PostgreSQL integration tests against a disposable schema. Extend fixtures for the two-ticket worked example, grouped archive failures, NOT_US, incomplete scripts, Windows/Linux filesystems, ignored/unconfigured resources, threshold boundaries and unsupported results. Add mailbox/retry/concurrency, timezone/DST/late-arrival, permission, recovery and browser workflow tests as those features are delivered. Record which checks actually ran; an RFC alone is not evidence that they passed.

## 11. Open decision register

| Decision required | Blocks |
| --- | --- |
| Mail provider, access method, polling/events, matching normalization and ambiguity handling | Live ingestion |
| Supported parser versions and mandatory pilot checks; real failing FRA/failed-job fixtures | Parser coverage and honest pilot acceptance |
| Provider/hash deduplication scope, retry/claim policy, raw MIME/source retention | Durable ingestion and concurrency |
| Window boundaries, grace, DST, administrator bounds, configuration edits, malformed and late-arrival linking | Background scheduling |
| Repeated-failure occurrence/merging policy | Any change from current per-report tickets |
| Canonical server/alias model and unmatched-host review | Complete execution identity snapshots |
| Unresolved/Resolved assessment semantics | Those assessment views |
| Database/ticket note edit permissions; rich text/media/avatar representation and storage | Remaining UI interactions |
| Browser token storage and recovery delivery | Browser authentication and recovery |
| Production database setup/migrations, storage, deployment, retention, monitoring, backups and performance targets | Persistent internal pilot and production planning |

## 12. Acceptance criteria

This RFC is ready to guide implementation when its proposed boundaries and sequencing have been reviewed and each blocking decision has an explicit disposition. Draft status does not mean all open choices are approved.

The **PoC pilot** must meet the [product acceptance criteria](documentation.md#16-proof-of-concept-acceptance-criteria), with evidence that:

1. A real configured daily-check email is durably ingested exactly once through retry/concurrency, matched correctly, and available as raw source plus immutable results/settings/server snapshots.
2. Selected checks produce the documented grouped tickets; ignored/unconfigured checks and NOT_US do not; incomplete output yields only its Missing Email issue; unsupported output is never shown as passed.
3. Expected windows are evaluated automatically with the agreed timezone/grace/DST rules and no duplicate absence issues; late arrivals follow the agreed linking policy without automatic closure.
4. Roles, fixed session deadlines, revocation and delivered password recovery work end to end.
5. The responsive UI supports the agreed dashboard, lists, lazy-loaded detail/raw source/timeline, comments and status transitions, notes, themes, avatars and administrator closure history.
6. Persistent Docker deployment survives restart and a tested restore; required parser, permission, schedule and core end-to-end checks pass, with realistic performance evidence.

SQL monitoring, standby alerts and authentication rate limiting remain Release 1.0 deliverables and must not be represented as completed PoC capabilities.
