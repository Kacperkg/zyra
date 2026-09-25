# Zyra

Zyra is a self-hosted monitoring and ticket-management application for Oracle database health-check emails. It receives assessment emails produced by existing server-side scripts, parses their contents, records every assessment, and raises actionable tickets when a check fails or an expected email is missing.

This document defines the initial product scope and technical design. The MVP is a proof of concept (PoC) used to validate the core Oracle daily-check workflow before Release 1.0. An initial Go API and React frontend exist. The frontend covers login, dashboard, Oracle ticket lists and ticket detail; live mailbox ingestion, other product screens, production deployment and several parser/infrastructure decisions remain incomplete.

## 1. Goals

Zyra should:

- turn Oracle health-check emails into structured assessments;
- identify failures such as backup, FRA, archive destination, filesystem, ASM, tablespace, and malformed or missing-email problems;
- create tickets only for enabled checks that breach the configured rules;
- preserve the full history of passed and failed assessments;
- let users investigate, discuss, close, and reopen tickets;
- provide fast navigation across clients, databases, assessments, and tickets;
- enforce role-based permissions;
- run as a self-hosted, Dockerized Go and React monorepo.

## 2. Scope

### 2.1 Proof-of-concept (MVP) scope

- Oracle daily-check emails, normally expected twice a day (AM and PM).
- Email ingestion into a dedicated Zyra mailbox.
- Subject/sender-based matching of emails to a configured database.
- Storage of the original email and parsed assessment result.
- Configurable check enablement and thresholds per database.
- Detection of malformed/incomplete assessment emails.
- Detection of an email not arriving within a configured schedule window.
- Ticket creation, commenting, closing, and reopening.
- Clients, databases, email-source configuration, schedules, assessments, users, and profiles.
- Admin, trusted, and normal-user roles.
- Light theme by default, with a dark-theme option.
- Docker-based self-hosting.

### 2.2 Release 1.0 scope

- SQL checks and SQL issue pages.
- The separate standby-alert email that currently runs every 30 minutes.

These two capabilities are intentionally excluded from the PoC so the daily-check workflow can be validated first. They are committed Release 1.0 scope rather than optional future ideas.

The PoC data model and parser boundaries should allow SQL and standby assessments to be added for Release 1.0 without redesigning tickets or assessment history.

### 2.3 Not currently planned for the PoC or Release 1.0

- Sending commands to, or making changes on, monitored database servers.
- Automatically fixing database problems.
- SaaS/multi-tenant hosting and billing.
- Native mobile applications.

## 3. Terminology

| Term | Meaning |
| --- | --- |
| Client | A customer or organisation that owns one or more databases. |
| Database | A monitored Oracle database belonging to a client. |
| Email source | Rules used to associate an incoming email with a database, including sender and subject. |
| Assessment | One received and processed health-check email, including its raw source and parsed results. |
| Check result | The outcome of one check within an assessment, such as Filesystem or Backups. |
| Ticket | An actionable issue created from a failed check, malformed email, or missing expected email. |
| Daily check | The current Oracle health assessment, normally received in AM and PM windows. |
| Missing email | Either no email arrived during an expected window, or an email arrived but could not be treated as a complete valid assessment. |

## 4. Users and permissions

Permissions must be enforced by the API, not only hidden in the UI.

| Capability | Normal | Trusted | Admin |
| --- | :---: | :---: | :---: |
| View dashboard, clients, databases, assessments, and tickets | Yes | Yes | Yes |
| Search and filter | Yes | Yes | Yes |
| Comment on tickets | Yes | Yes | Yes |
| Close and reopen tickets | Yes | Yes | Yes |
| Edit own profile and profile picture | Yes | Yes | Yes |
| Add and edit clients/databases | No | Yes | Yes |
| Edit email sources, schedules, enabled checks, and thresholds | No | Yes | Yes |
| Delete clients/databases/configuration | No | No | Yes |
| Manage users and roles | No | No | Yes |
| View another user's ticket-closure activity | No | No | Yes |

Deletion should preferably be a reversible archive/soft-delete operation so historical assessments and tickets remain intact.

## 5. Primary workflow

1. An existing script runs an Oracle database health check.
2. The script emails its report to the existing operational address.
3. A mail rule forwards relevant messages to a dedicated Zyra mailbox. This isolates ingestion from the human inbox and follows the recommended client/event model.
4. A Zyra email-ingestion worker connects to that mailbox, retrieves the message, and stores the raw email before parsing it.
5. Zyra matches the sender and normalized subject to a configured database and assessment type.
6. The appropriate versioned parser converts the body into structured check results.
7. Zyra evaluates enabled checks and database-specific thresholds.
8. The assessment and all check results are stored whether they passed or failed.
9. Failed results create or update tickets according to the deduplication policy.
10. Users investigate the ticket, leave comments, and close it. A closed ticket can be reopened at any time.

### 5.1 Recommended ingestion approach

For the PoC, relevant messages should be forwarded to a dedicated Zyra email address and the application should act as a client of that inbox. The exact mail provider, protocol/API, and whether ingestion uses polling or events have not been decided.

Each message must be idempotent. Zyra should store the provider message ID and a deterministic content hash, and must not create a second assessment or ticket when the same email is forwarded or fetched twice.

A temporary parsing or mailbox failure must not silently lose an email. The implementation approach for retries and failed processing has not been decided.

## 6. Email matching and parsing

### 6.1 Email-source configuration

Each database may have one or more assessment email sources containing:

- expected sender address;
- expected subject or an explicit safe matching pattern;
- assessment type, initially `daily_check`;
- parser type and parser version;
- timezone;
- enabled/disabled state;
- one or more expected schedule windows;
- optional grace period.

Example subject: `XXXX_XXX IFSPRD - Daily Check`.

Matching should normalize harmless differences such as leading/trailing whitespace and repeated forwarding prefixes (`FW:`, `Fwd:`), but should not use broad fuzzy matching that could assign a report to the wrong database.

Unmatched email should be retained in an ingestion-review queue for an administrator; it should never be discarded.

### 6.2 Parser contract

A parser receives the immutable raw message and returns:

- database identity discovered in the report, if present;
- execution hostname and IP address, if present or resolved through configured server metadata;
- assessment timestamp and type;
- parser version;
- overall parsing status;
- a collection of normalized check results;
- warnings for missing, truncated, duplicated, or unrecognised sections.

Each check result contains:

- stable check type;
- display title;
- result: `passed`, `failed`, `warning`, `ignored`, `not_evaluated`, or `unknown`;
- measured value(s) and unit, where applicable;
- configured threshold used for evaluation;
- short summary suitable for a ticket title;
- relevant raw excerpt or structured evidence.

The parser extracts facts; the rule evaluator decides whether those facts create a ticket. Keeping these separate makes parser testing and future rule changes safer.

The daily-check parser must also handle the format shown in the first supplied anonymised email sample:

- mail-system warning banners may appear before the report and must be ignored as transport content, not interpreted as part of the assessment;
- `ReportOn`, `PkgVersion`, `UniqueID`, `Database`, database version, and script information are report metadata;
- checks are introduced by stable labels such as `Datafiles=`, `Backups=`, and `Tablespaces=`;
- `OK!` means the report found no issue for that check, but an enabled/ignored rule must still be applied and stored;
- `Backups=NOT_US` means the company using Zyra is not responsible for that database's backups. It is a non-failure business state and never creates a Backups ticket;
- multi-line sections use trailing `\` characters as formatting/continuation markers; these must not become part of parsed values;
- tabular sections must be parsed into individual resource findings rather than stored only as one block of text;
- the `=@=` marker terminates the database-statistics block before the filesystem section;
- Windows paths, spacing, case differences, old Oracle versions, and forwarded email formatting must be preserved or normalized safely without breaking section recognition;
- `ReportOn` is the assessment time. The received-at time is stored separately.

Daily checks may run on Windows or Linux. Accept both Windows `Filesystem Usage` drive rows and Linux `Filesystem Mounted Use%` tables. Linux filesystem rules identify resources by mount point, preserving the device/source in evidence. Linux reports may end with `Script`, `Version`, `Run by`, and schedule metadata without a `Script Info` heading; this is a valid layout, not itself evidence of a missing or incomplete email. Required database-check content must still be present.

For this report format, `Script Info` identifies where the check ran. From a value such as `Run by: username@hostname`, the parser must extract only the portion after the final `@` as the observed execution hostname; the username is not part of the connection target. Zyra then matches the observed hostname to a configured database server record containing a canonical hostname and optional IP address. Both the observed value and the matched server identity must be stored. If no server matches, the assessment remains processable but shows an explicit `Unmatched host` warning for administrative review rather than silently attaching an incorrect IP.

Zyra should not depend on live DNS lookup when a user opens a ticket. Hostname and IP displayed on an assessment or ticket should be immutable snapshots taken when the assessment is processed, with links to the current server configuration. This ensures an older ticket still shows the original connection target after a server migration or IP change.

An external-email caution banner is untrusted message content. It is neither an instruction to Zyra nor evidence of a database failure.

### 6.3 Initial check catalogue

The initial Oracle catalogue should support:

- ASM space;
- archive destinations;
- backups;
- clusterware;
- datafiles;
- extents;
- failed jobs;
- filesystem;
- indexes;
- invalid objects;
- max lag;
- recovery area/FRA space;
- segments;
- tablespaces;
- malformed/incomplete email;
- missing expected email.

Check identifiers should be machine-friendly stable values such as `archive_destinations` and `filesystem`; display names can change without breaking history.

### 6.4 Worked parsing example

The first anonymised sample is a daily check produced by package version `2.5` for database `ifsprd`. It contains, among other data:

```text
Backups=
Datafiles needing backup:
RMAN  E:\ORADATA\IFSPRD\APEX01.DBF  22-SEP-2026 00:42:16
...

Tablespaces=
Name       Total MB  Max. MB  Used MB  %Free
UNDOTBS1   31744     31744    30900     2.7%
```

Each database controls which sections of the email Zyra evaluates through its selected and excluded checks. For this database, Backups and Tablespaces are selected for evaluation. The same email therefore produces exactly two tickets:

| Ticket | Parsed evidence | Ticketing rule |
| --- | --- | --- |
| Backups | The `Backups` section contains multiple datafiles needing backup and their last-completed timestamps. | The Backups check is selected and the non-empty failure list breaches its rule. Create **one grouped Backups ticket** for the assessment, with every affected file attached as structured evidence. Never create one ticket per datafile from this section. |
| Tablespace (`UNDOTBS1`) | `UNDOTBS1` has `2.7%` free in the `Tablespaces` section. | This is a Tablespace check. The configured minimum-free threshold is breached, so create one Tablespace ticket for resource `UNDOTBS1`. `UNDO` is part of the affected tablespace's name, not a separate check type. |

The email also contains unusable indexes and filesystem usage values, including drive `E:` at `91%`. Indexes and Filesystem are **not selected/configured** for this database example, so their sections are not evaluated and cannot create tickets. Their contents remain available in the immutable raw email; Zyra may record only that the sections were skipped as `not_evaluated`. Once an administrator or trusted user selects and configures those checks, later assessments can evaluate them. Sections containing `OK!` count as passed only when that check is selected for evaluation. Therefore, the presence of a non-empty section alone is not a universal ticket rule.

In summary, the expected outcome for this assessment is exactly:

1. One Tablespace ticket for `UNDOTBS1` at `2.7%` free.
2. One Backups ticket containing the complete list of affected datafiles.

No additional ticket is created for each backup row, and no Indexes or Filesystem ticket is created because those checks are not selected for this database.

The expected high-level parser output is:

```json
{
  "assessmentType": "daily_check",
  "database": "ifsprd",
  "reportTime": "2026-09-23T11:27:59",
  "packageVersion": "2.5",
  "ticketCandidates": [
    { "checkType": "backups", "resources": "parsed from all backup rows" },
    { "checkType": "tablespace", "resource": "UNDOTBS1", "freePercent": 2.7 }
  ],
  "ticketsCreated": 2
}
```

The Backups candidate above is an abbreviated representation; the actual output contains the structured method, path, and last-completed time from every parsed row. Database timezone must be applied before converting `ReportOn` to a UTC timestamp.

### 6.5 Missing and malformed emails

Zyra must distinguish:

1. **Not received:** no matching valid email arrived during the configured window plus grace period.
2. **Received but malformed:** an email arrived, but required sections were absent, truncated, or unparsable.

The missing-email scheduler evaluates every enabled email source in its configured timezone. A schedule can define separate AM and PM windows, for example `18:30–22:00`. A valid assessment satisfies only the appropriate window. Each missed window creates at most one ticket.

If a delayed valid email arrives after a missing-email ticket was created, Zyra should link the assessment to that ticket and mark it as no longer missing. Automatic closure is a product decision still to be confirmed; the PoC default is to leave the ticket open for a user to review.

#### 6.5.1 Received-but-incomplete example

The supplied Missing Email example is an email that arrived, but the daily-check script failed before producing a complete assessment. Its body begins with a PL/SQL failure and then contains only the later filesystem and script-information sections:

```text
ERROR at line 3:
ORA-06550: line 3, column 17:
PLS-00201: identifier 'XXXDCX.XXXX' must be declared
ORA-06550: line 2, column 4:
PL/SQL: Statement ignored

Filesystem Usage
...

Script Info
...
```

This must create one **Missing Email** ticket with a reason such as `Received but incomplete`. It is not the same as the scheduled `Not received` case.

The ticket evidence should include:

- the Oracle errors (`ORA-06550` and `PLS-00201`);
- the incomplete or absent daily-check header and required database-check sections;
- the received time;
- the script path;
- the execution hostname parsed from `Run by`;
- the original body in the ticket's Raw Email tab.

Completeness is determined using the expected structure for the matched email source and parser version. A trailing `Filesystem Usage` or `Script Info` section does not make the assessment valid when the main database-check output is missing. Partial check data in an incomplete email must not create ordinary check tickets, because the report did not complete reliably. The one Missing Email ticket represents the failed assessment run.

The parser must tolerate formatting introduced by email rendering or copying, including bold markers, escaped underscores and backslashes, tabs, non-breaking spaces, and HTML character entities. These presentation characters are not part of Oracle identifiers, paths, hostnames, or error codes.

### 6.6 Archive destinations example

The supplied package-version `2.5` daily check contains a failed `ArchiveDestinations` section:

```text
ArchiveDestinations=
Invalid Archive Destinations:
Dest.Id    Destination    Status    Error
2          ERROR          ifsd_stby ORA-03135: connectio...
```

When Archive Destinations is selected for that database, the non-empty invalid-destinations table creates one **Archive Destinations** ticket. All failing destination rows from the assessment belong to that single ticket rather than producing one ticket per row.

The ticket should include the destination ID, reported destination/status fields, complete Oracle error text available in the raw message, database, assessment time, hostname/IP, and a Raw Email tab. The supplied pasted formatting may not preserve the original fixed-width column alignment, so the parser fixture must be built from the original raw email before finalising the exact column mapping; Zyra must not silently swap the destination and status values.

Other sections in this email show `OK!`; they are recorded as passed only when they are selected for this database. `Backups=NOT_US` means backups for this database are not the responsibility of the company using Zyra. The assessment should display this as `Not managed by us` (or equivalent wording), and it must not create a Backups ticket. Filesystem results create tickets only if Filesystem is selected/configured for this database and its rules are breached.

## 7. Check configuration and thresholds

Configuration is per database. Each known check is selected or excluded. Only selected checks are evaluated against their rules and can create tickets. An excluded or not-yet-configured section is skipped; its content remains available in the raw email but does not need to be parsed into resource-level results.

Initial rule shapes include:

| Check | Configuration |
| --- | --- |
| Filesystem | Filesystem/mount name, enabled or ignored, maximum usage percentage. |
| Tablespace | Tablespace name, enabled or ignored, minimum required free percentage. |
| ASM space | Disk group/name, enabled or ignored, minimum required free percentage. |
| Backup | Backup target/name, enabled or ignored, parser-specific success criteria, and support for `NOT_US` when backups are outside the company's responsibility. |
| Archive destinations | Destination, enabled or ignored, and acceptable status. |
| FRA/recovery area | Enabled or ignored and maximum usage/minimum free threshold. |
| Missing email | Expected schedule window, timezone, grace period, enabled or ignored. |

TODO — Recovery Area Space: await a real failing email example before finalising non-OK parsing, failure criteria, and ticket/system-comment details. `RecoveryAreaSpace=OK!` means passed when selected; other output is likely an issue but is not yet a confirmed blanket ticket-creation rule. Keep the current non-OK result as `unknown` until this is fleshed out.

Checks or resources without matching configuration should be skipped as `not_evaluated` and may be displayed for configuration review. Zyra must not silently choose a threshold or create a ticket for an unconfigured check, mount, tablespace, disk group, or backup target.

An assessment should retain the effective rule values used to calculate its result so historical outcomes do not change when thresholds are edited later. Any wider change-history or audit requirements have not been decided.

## 8. Tickets

### 8.1 Ticket fields

- unique numeric ticket number;
- status: `open` or `closed` initially;
- check/issue type;
- assessment type;
- title and summary;
- client, database, and execution server snapshot, including observed hostname, canonical hostname, and IP address when configured;
- source assessment and check result, where applicable;
- created and updated timestamps;
- closed timestamp and closing user;
- reopen timestamp/history;
- comments and participants;
- links to up to five recent similar issues.

Suggested title format: `Backups check failed for <client> / <database> – Daily Check`.

For the worked email example, suitable titles are:

- `Backups check failed for <client> / IFSPRD – Daily Check`
- `Tablespace check failed for <client> / IFSPRD – UNDOTBS1 at 2.7% free`
- `Missing email for <client> / <database> – Received but incomplete`
- `Archive Destinations check failed for <client> / IFSDCDB – Daily Check`

### 8.2 Creation and deduplication

The cross-assessment repeat-failure policy is unresolved. The current API creates tickets for each distinct report and does not merge later failures into an existing open ticket. Before changing that behaviour, decide whether a repeated failure should append an occurrence to an open ticket or create a new ticket. Do not present either option as the confirmed PoC policy.

Regardless of that later decision, repeated ingestion of the same email must remain idempotent, unrelated resources must not be merged solely because their check type matches, and previous tickets remain available through Recent similar issues.

### 8.3 Ticket list

Open and Closed Oracle issues are separate views, reached through the Issues navbar dropdown or dashboard buttons. The default operational view contains only open tickets. Do not include an All/Open/Closed switch on the issues page itself. Each view supports:

- filters for client, database, ticket/check type, created date, and assessment type, within the selected status view;
- sorting by ticket number, check type, created date/time, client, database, and assessment type;
- 50 tickets initially, another 50 loaded automatically when scrolling near the bottom, then Next page after at most 100 tickets; each new page starts with 50 again (or fewer if fewer remain). No Load more button. Do not request more once the results are exhausted;
- a clear empty state and loading state;
- persistent filters in the URL so a view can be bookmarked or shared.

Default column order:

1. Ticket number
2. Check type
3. Created date
4. Created time
5. Client/customer
6. Database
7. Assessment type
8. Status

Closed tickets remain searchable and visible.

Ticket lists fetch summary fields only: identifiers, check type, created timestamp, client/database display information, assessment type, and status. Do not preload comments, full findings/evidence, or raw email bodies. Selecting a row navigates to a ticket detail route such as `/tickets/{ticketId}` and loads its detail data. The 100-row UI page and 50-row fetch batch are distinct; preserve stable ordering across batches.

### 8.4 Ticket detail

The ticket page should show:

- issue title, status, ticket number, client, database, execution hostname, IP address, and timestamps;
- a link to the source assessment and Discussion/Raw Email tabs directly below the title and metadata;
- parsed evidence and all linked occurrences;
- chronological system events and user comments;
- `Comment` and `Comment and close` actions;
- a `Reopen` action for closed tickets;
- a right-hand context panel containing notes, linked client, linked database, participants, and up to five similar issues ordered as described below.

**Discussion** is the default tab. Its first timeline entry is one system-generated comment describing the findings for this ticket, not merely saying a check failed. Keep multiple findings belonging to the ticket inside that single comment, using a compact list or table:

- Backups: describe the reported problem and affected targets/datafiles. Mention newly added datafiles only when the report explicitly identifies them; do not infer a cause.
- Tablespaces: identify affected names, reported percentages clearly labelled free or used, and the configured thresholds.
- Filesystem: identify Windows disks or Linux mounts, usage percentages and configured maximums; retain device/source details where available.
- Other checks: identify affected resources and relevant reported evidence. Distinguish absent emails from received-but-incomplete reports.

Findings and effective thresholds are snapshots from assessment processing, not values recalculated from current configuration. Failed results from one assessment are grouped into one ticket per check type, so all backup flags, affected tablespaces, or affected filesystem resources for that check appear together in its single system findings event. User comments and status changes follow chronologically. Request 50 timeline events initially, another 50 automatically on scroll, then move to the next 100-event UI page rather than embedding an unbounded timeline in ticket detail.

**Raw Email** displays the original body as read-only text with preserved whitespace/line breaks and horizontal scrolling when needed. Show subject, sender and received time above it. Load the body when the tab opens; do not use a separate raw-email link or open a separate page. Treat content as text, never executable HTML. When no email arrived, show an explicit No email received state rather than an empty viewer.

Keep the ticket title, metadata and context panel visible across both tabs. Move the context panel below the main content on mobile.

**Similar issues** contains up to five tickets matching the same client, database and check/issue type, excluding the current ticket. Pin the most recently created open matching ticket first, if one exists. Fill the remaining slots with the most recently created matching tickets regardless of status, excluding the pinned ticket. If no open match exists, show the five most recent matches. Matches may be newer or older than the current ticket. Order equal creation timestamps by ticket ID descending for deterministic results; return fewer than five when fewer matches exist.

Each entry links to that ticket and displays its number, creation date/time and Open/Closed status. For example, an unresolved morning Tablespace ticket appears first when investigating a new Tablespace ticket for that same client/database, helping the operator assess whether they concern the same problem. A matching check type alone does not prove identical findings. This navigation rule does not merge tickets or change their status. Automatic merging/closure remains deferred, and a manual merge action has not been specified. The API implements this selection in the ticket detail response using a single database query.

Comments should support a controlled rich-text subset: paragraphs, headings/body sizes, bold, italic, underline, lists, alignment, links, text colour, images, and GIFs. Content must be sanitized on the server. Uploaded files require type/size limits and should be served from authenticated storage; arbitrary embedded HTML or JavaScript is not allowed.

Every status change is an immutable timeline event showing who performed it and when.

Confirmed ticket interaction decisions:

- Show separately labelled **Database notes** and **Ticket notes**. Database notes apply across the database's tickets; ticket notes apply only to that issue. Editing one must not overwrite the other. Editing permissions remain to be specified.
- Allow comments on closed tickets without reopening them. Reopening requires the explicit Reopen action.
- **Comment and close** requires a non-empty comment and saves the comment and closure atomically. Closing without a comment, if offered, must use a separate **Close** action rather than submitting an empty Comment and close.
- Automatic ticket closure is deferred pending further design. For now, a passing assessment or late-arriving email must not automatically close an existing ticket. Cross-report failure grouping remains a separate unresolved decision.

## 9. Screens and navigation

### 9.1 Shared navigation

Use the same responsive top navbar throughout the application, with no sidebar:

- Dashboard.
- Issues: a two-column dropdown titled Oracle and SQL, each containing Open and Closed.
- Assessments: a two-column dropdown titled Oracle and SQL, each containing Unresolved, Resolved, Passed, Failed and All.
- Clients.
- Settings & maintenance: a placeholder for later; its contents are outside the PoC.
- Theme control and avatar/profile button at the top-right.

SQL options remain visibly unavailable during the PoC. The assessment dropdown labels are agreed, but the exact Unresolved/Resolved semantics remain pending. Proposed interpretation: Passed/Failed describes the assessment result, while Unresolved/Resolved reflects whether associated tickets still require attention. Do not implement that proposal as a settled rule. Closing tickets never changes the historical assessment result.

Only one navbar dropdown may be open at a time. Opening another replaces the active dropdown; selecting a navigation link, clicking outside, or pressing Escape closes it. Use a subtle fade and short vertical transition, respecting reduced-motion preferences. Triggers expose their expanded state and support keyboard use; Escape returns focus to the triggering control. A shared dropdown panel spanning the navbar width is a candidate for the next mockup, not a finalized width requirement. Keep its Oracle/SQL columns compact and aligned with the navigation.

### 9.2 Dashboard

The dashboard contains only two equal columns, Oracle and SQL, stacked on mobile. Oracle shows its open-ticket count with separate Open and Closed navigation buttons. SQL mirrors this layout as a clearly unavailable placeholder, with navigation disabled until Release 1.0. Do not imply that SQL monitoring is active. No charts, activity feeds, extra statistics or widgets.

### 9.3 Client page

One client record can contain both Oracle and SQL databases. Identify the engine for each database and integrate both into the client's database list; do not create separate client records per engine. Actual SQL monitoring remains Release 1.0 scope.

- client details;
- searchable list of configured databases;
- links to each database;
- add/edit actions for trusted users and admins;
- archive/delete action for admins only.

### 9.4 Database page

- database identity, configured server hostname(s), IP address(es), client, and status;
- enabled/excluded check configuration;
- check-specific threshold tables;
- assessment email sources and schedules;
- assessment history with date, type, and passed/failed result;
- filters for assessment history;
- participants who have commented on this database's tickets;
- five recent similar/relevant issues.

Selecting an assessment email source opens a detail page or panel where trusted users and admins can edit its sender, subject rule, parser, and expected schedules.

### 9.5 User profile and administration

Users can change their own display details, password, theme, and profile picture. Admins can list users, manage roles/status, and view the tickets closed by a selected user.

### 9.6 Visual direction

Light is the default theme and green is the sole accent colour. Dark mode uses neutral charcoal surfaces, subtle grey borders and soft white text, not green-tinted backgrounds. Reserve green primarily for links and primary actions. Use restrained typography, spacing and decoration; avoid gradients, oversized rounded cards and excessive shadows. Use the approved system sans-serif stack (`-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, `sans-serif`), with a system monospace stack for technical values. Mockup data and interactions illustrate the design, not implemented or live functionality.

Define shared CSS custom properties in `src/styles/tokens.css`; component/page CSS modules consume them instead of repeating literal colours and radii. `ThemeContext` sets `data-theme="light"` or `data-theme="dark"` on the root HTML element. Theme preference persistence remains an implementation decision.

| Token | Light | Neutral dark |
| --- | --- | --- |
| `--color-background` | `#f6f7f6` | `#171717` |
| `--color-surface` | `#ffffff` | `#222222` |
| `--color-surface-subtle` | `#eaf3ed` | `#2c2c2c` |
| `--color-text` | `#26332b` | `#e8e8e8` |
| `--color-text-muted` | `#69766d` | `#aaaaaa` |
| `--color-border` | `#dde4df` | `#393939` |
| `--color-accent` | `#176e43` | `#71c795` |
| `--color-on-accent` | `#ffffff` | `#172019` |
| `--color-focus-ring` | `rgba(23, 110, 67, 0.25)` | `rgba(113, 199, 149, 0.3)` |
| `--shadow-dropdown` | `0 8px 24px rgba(0, 0, 0, 0.07)` | `0 8px 24px rgba(0, 0, 0, 0.22)` |

Shared geometry tokens: `--radius-small: 3px`, `--radius-control: 4px`, `--radius-panel: 5px`, `--radius-shell: 7px`, `--radius-round: 999px`, and `--content-width: 1200px`. Motion tokens: `--transition-fast: 120ms ease` and `--transition-standard: 180ms ease`. Disable nonessential motion when reduced motion is requested. Validate contrast and keyboard focus visibility during frontend implementation.

## 10. Authentication and account recovery

- Access tokens use JWT (JSON Web Token) and expire after 15 minutes.
- Refresh sessions expire seven days after sign-in. Refreshing access tokens does not extend this deadline; the user must sign in again when it is reached.
- While the refresh session is valid, the application can obtain a new access token without requiring the user to sign in again.
- Password-recovery support for user accounts.

The initial API uses the approved JWT/bcrypt approach: HS256 JWTs via golang-jwt/jwt/v5, hashed random refresh tokens rotated on use, and server-side session revocation checks. The initial frontend keeps tokens in memory: a full reload requires signing in again. Persistent browser token storage and recovery-email delivery remain open. Any access token issued near the seven-day session deadline must expire no later than that deadline so access cannot continue beyond it without signing in again.

All roles use the same login screen. There is no public registration screen; administrators provision accounts. Superadmin is a possible later addition and is not included in this PoC. For local testing only, the API supports an opt-in `admin/admin` fixture using the existing admin role. It requires development mode and a loopback listener; ordinary password validation remains unchanged. See the [local account setup](zyra-api/README.md#local-frontend-test-account-and-sample-data). Turning the fixture flag off denies its login, access and refresh.

## 11. Proposed architecture

The initial Go API is implemented using the approved Gin and GORM/PostgreSQL stack. See the [API contract](zyra-api/doc/API.md) for working routes, run instructions, limitations, and remaining work.

### 11.1 Monorepo layout

```text
zyra/
├── zyra-api/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── doc/
│   └── internal/
│       ├── apperrors/
│       ├── auth/
│       ├── config/
│       ├── database/
│       ├── handlers/
│       ├── middleware/
│       ├── models/
│       ├── repository/
│       ├── routes/
│       └── services/
└── zyra-web/
    └── src/
        ├── api/
        ├── assets/
        ├── components/
        │   ├── layout/
        │   ├── navigation/
        │   ├── tickets/
        │   └── ui/
        ├── contexts/
        ├── hooks/
        ├── pages/
        ├── routes/
        ├── styles/
        │   ├── tokens.css
        │   ├── reset.css
        │   └── globals.css
        ├── types/
        ├── utils/
        ├── main.tsx
        ├── router.tsx
        └── routeTree.gen.ts
```

Zyra will be a monorepo. The Go API lives in `zyra-api/`, while the React application lives in `zyra-web/` with its application source under `zyra-web/src/`.

Within the API layers, use domain-specific files wherever applicable: for example, `models/ticket_model.go`, `repository/ticket_repository.go`, `services/ticket_service.go`, and `handlers/ticket_handler.go`. Apply the same convention to other domains, keeping shared infrastructure/helpers shared and omitting layers a feature does not need.

### 11.2 Components

- **`zyra-api`:** Go API.
- **`zyra-web`:** React application built with Vite, using TanStack Router, React Context and CSS Modules.
- **Development database:** PostgreSQL.

The production database setup, background-processing model, file and email storage approach, caching, queues, and other supporting infrastructure have not been decided.

### 11.3 Frontend organization

The initial frontend follows this organization. See [zyra-web/README.md](zyra-web/README.md) for setup, verification and current limitations. The directory diagram describes responsibilities; create only folders used by implemented features.

- `api/`: shared HTTP client and domain endpoint modules such as `tickets.api.ts`.
- `routes/`: thin TanStack Router route definitions; screen composition belongs in `pages/`. `routeTree.gen.ts` is generated by routing tooling.
- `pages/`: domain folders such as `tickets/`, `clients/`, `databases/`, `assessments/`, `dashboard/` and `auth/`; each page imports its own colocated `.module.css`.
- `components/`: reusable UI under `ui/`, shell/navigation under their respective folders, and domain components under folders such as `tickets/`. Larger components can own a folder containing their component, CSS module and related private components.
- `contexts/`: focused `AuthContext` and `ThemeContext`; use React Context rather than Redux. Keep forms, filters and other page-local state local, and avoid copying all server data into a global context. Server-data fetching/cache strategy remains open.
- `hooks/`, `types/`, `utils/` and `assets/`: shared hooks, API/domain types, utilities and bundled assets respectively.
- `styles/`: global tokens, browser reset and base typography only. Use CSS Modules for component/page styling; do not add Tailwind.

Create domain folders as their features are introduced rather than scaffolding empty future areas. Authentication context must preserve the 15-minute access/fixed seven-day session contract. The initial in-memory session uses a single coordinated refresh request for concurrent API calls and rejects stale responses after logout or account changes. A persistent browser storage mechanism has not been selected.

## 12. Core data model

The following entities describe the information Zyra is expected to manage. This is a conceptual model, not a final PostgreSQL schema; table names, relationships, and fields will be decided during implementation.

- `users`, `roles`, `user_roles`, `refresh_sessions`, `password_reset_tokens`
- `clients`
- `databases`
- `database_servers` (canonical hostname, optional IP address, aliases, active state, and connection/display notes)
- `assessment_email_sources`
- `assessment_schedules`
- `check_definitions`
- `database_check_settings`
- `resource_thresholds`
- `raw_emails`
- `assessments`
- `assessment_check_results`
- `tickets`
- `ticket_occurrences`
- `ticket_comments`
- `ticket_events`
- `attachments`

Important integrity rules:

- client/database names may repeat globally but should be unique within their appropriate parent scope;
- a raw email provider ID and content hash support idempotency;
- an expected schedule window can produce at most one missing-email occurrence;
- an assessment stores the observed execution hostname plus the matched server hostname/IP snapshot used by its tickets;
- assessments and check results are append-only operational history;
- tickets and comments must not be cascade-deleted when a user is disabled;
- stored timestamps use UTC, while schedules retain an IANA timezone such as `Europe/London`.

## 13. API outline

See the [API contract](zyra-api/doc/API.md) for implemented operations and remaining work.

The following remains the product-level operation outline. For exact implemented methods, `/api` paths, payloads, and deferred operations, use the [API contract](zyra-api/doc/API.md).

```text
POST   /auth/login
POST   /auth/refresh
POST   /auth/logout
POST   /auth/forgot-password
POST   /auth/reset-password

GET    /dashboard
GET    /tickets
GET    /tickets/{ticketId}
POST   /tickets/{ticketId}/comments
POST   /tickets/{ticketId}/close
POST   /tickets/{ticketId}/reopen

GET    /clients
POST   /clients
GET    /clients/{clientId}
PATCH  /clients/{clientId}

GET    /databases
POST   /databases
GET    /databases/{databaseId}
PATCH  /databases/{databaseId}
GET    /databases/{databaseId}/assessments
GET    /databases/{databaseId}/check-settings
PUT    /databases/{databaseId}/check-settings

GET    /assessments/{assessmentId}
GET    /assessments/{assessmentId}/raw-email

GET    /users/me
PATCH  /users/me
POST   /users/me/avatar
GET    /admin/users
GET    /admin/users/{userId}/closed-tickets
```

The initial API uses page/limit pagination with allowlisted filtering and stable sorting. The API contract lists the currently supported filters.

## 14. Performance and usability

Loading speed is a primary requirement. Initial targets:

- pages, ticket lists, and searches should feel fast during normal use;
- large ticket, assessment, comment, and history lists must remain usable;
- loading and empty states should be clear;
- responsive layouts and keyboard-accessible controls;
- readable colour contrast in both themes.

The ticket loading interaction is agreed: 50-row batches, automatic scroll loading up to 100 rows per UI page, then explicit pagination. List responses contain summaries only; details, timeline pages, and raw email load separately. Initial PostgreSQL indexes cover the implemented status/date, client/database/check/assessment-type, closure-history, assessment-history/schedule, similar-ticket and timeline query paths. Exact performance targets, search optimisation, database pagination strategy, caching and other frontend optimisations remain open and should be validated against realistic volumes.

## 15. Testing strategy

- Unit tests for every parser section and rule evaluator.
- Golden-file tests using anonymised real email fixtures.
- A golden-file test for the supplied package-version `2.5` daily check that asserts two tickets only: Backups and Tablespace (`UNDOTBS1`).
- Assertions that the same fixture skips Indexes and Filesystem as `not_evaluated`, without resource-level evaluation or tickets, because those checks are not selected/configured in this example.
- An assertion that all backup rows are grouped into one Backups ticket rather than creating a ticket for each affected datafile.
- Assertions that the external-email caution banner, line-continuation characters, and `=@=` delimiter do not corrupt report metadata or check sections.
- Assertions that `Run by: user@hostname` produces the observed hostname, matches the correct configured database server, and snapshots its canonical hostname/IP onto both created tickets.
- A golden-file test for the received-but-incomplete email that creates one Missing Email ticket, captures the Oracle errors and server details, and does not create tickets from the partial filesystem data.
- A golden-file test for the Archive Destinations email that groups all invalid destination rows into one Archive Destinations ticket when that check is selected.
- An assertion that `Backups=NOT_US` is shown as not managed by the company and never creates a Backups ticket.
- Tests for rendered/copied email artefacts, including HTML entities, non-breaking spaces, tabs, bold markers, and escaped punctuation.
- Tests for truncated, reordered, duplicated, forwarded, HTML-only, and unexpected email bodies. A Windows report ending at the literal `Script Info` heading without an actual `Run by` footer is incomplete; valid Windows and Linux footer layouts remain accepted.
- Schedule tests across timezones, daylight-saving changes, grace periods, and late arrivals.
- Idempotency and concurrency tests for duplicated messages and workers.
- Permission tests for every protected API operation.
- Integration tests using PostgreSQL and a test mailbox/email fixture source.
- End-to-end tests for login, ticket filtering, commenting, close/reopen, database configuration, and password recovery.
- Authentication, permission, and untrusted-content tests appropriate to the implementation choices made later.

Production parser fixtures must be anonymised and must not contain client credentials or sensitive database information.

The API includes an isolated Docker Compose PostgreSQL service for local integration testing. It binds to localhost, uses test-only credentials and temporary storage, and is not the persistent self-hosted deployment design.

## 16. Proof-of-concept acceptance criteria

The PoC is ready for an internal pilot when:

1. An authorised user can sign in, refresh a session, log out, and recover a password.
   Access tokens expire after 15 minutes (or earlier at the session deadline). Refreshing does not extend the fixed seven-day session; after that deadline, the user must sign in again.
2. Admins can manage users/roles; trusted users can configure clients and databases without deleting them; normal users have read-only configuration access.
3. A configured daily-check email is ingested exactly once, matched to the correct database, and stored with its raw source.
4. Supported checks are parsed into durable results and evaluated using the database's effective configuration.
5. Passed and failed assessments remain visible in history.
6. Enabled failures create correctly titled tickets; ignored checks do not.
   The supplied anonymised example creates exactly one Backups ticket and one Tablespace ticket for `UNDOTBS1`. Its unselected Indexes and Filesystem sections are skipped as `not_evaluated`; their content remains available in the raw email, and they create no tickets.
7. A missing or malformed email creates the appropriate ticket only once per expected window/message.
   The supplied incomplete-script example creates one Missing Email ticket with the Oracle failure as evidence and does not create ordinary tickets from its partial body.
   The supplied archive-destination example creates one grouped Archive Destinations ticket when that check is selected, regardless of how many invalid destination rows it contains. Its `Backups=NOT_US` value creates no Backups ticket.
8. Users can filter and sort Oracle tickets, view an issue timeline, comment, comment-and-close, close, and reopen.
9. Ticket pages show a system findings comment, Discussion/Raw Email tabs, client/database/source-assessment links, participants, separate database/ticket notes, and up to five linked previous issues matching the same client/database/check type.
   They also show the execution hostname and configured IP address captured when the assessment was processed.
10. Admins can see which tickets a user closed.
11. Light and dark themes work, with light as the default, and users can update their profile picture.
12. The application can be run as a self-hosted Dockerized project and preserves its core data. The detailed deployment and operational requirements remain undecided.
13. Permission, parser, schedule, and core end-to-end tests pass.

## 17. Proposed delivery phases

### Phase 0 — Discovery and fixtures

- Collect anonymised examples of successful and failed daily-check emails.
- Confirm mail provider and supported access method.
- Confirm exact subject/sender patterns and AM/PM schedules.
- Define the first parser grammar and check catalogue.

### Phase 1 — Foundation

- Monorepo, Docker development environment, PostgreSQL migrations.
- Authentication, refresh sessions, password recovery, roles, and users.
- Client/database configuration and basic responsive application shell.

### Phase 2 — Assessment pipeline

- Dedicated mailbox ingestion, immutable raw storage, matching, idempotency, and retry states.
- Daily-check parser, rule evaluation, check settings, thresholds, and assessment history.
- Missing-email scheduler.

### Phase 3 — Ticket workflow

- Dashboard, Oracle issue list, filters/sorting, ticket detail, comments, close/reopen, participants, and similar issues.
- Search, profile pictures, theme preference, and admin closure reporting.

### Phase 4 — Hardening and pilot

- Resolve the outstanding security, privacy, deployment, operations, and performance decisions; expand parser fixtures; and prepare the internal pilot.

### Release 1.0 — Committed expansion

- Add SQL checks and SQL issue pages.
- Add ingestion and ticket behaviour for the standby-alert email that runs every 30 minutes.
- Add authentication rate limiting using limits and shared-state infrastructure selected during Release 1.0 design; it is not part of the PoC implementation.
- Reuse the proven client, database, assessment, ticket, comment, role, and history workflows from the PoC.

## 18. Decisions needed before implementation

1. Which mail provider hosts the dedicated inbox, and should the PoC use IMAP, Microsoft Graph, Gmail API, or another supported API?
2. What are the exact AM/PM schedule windows, timezone, and allowed grace periods for each database?
3. What real email formats and script versions must the first parser support?
4. Which checks are mandatory for the first pilot, and what are their precise pass/fail rules?
5. Automatic closure is deferred: what recovery and late-arrival behaviour should be designed later? No automatic closure for now.
6. Should repeated failures update one open ticket or create a ticket per assessment? Neither option is currently the confirmed future policy.
7. Can normal users close/reopen tickets, or should that be limited to trusted users and admins?
8. Who may edit database-wide notes and ticket-specific notes? Their separate scopes are agreed.
9. What retention period and access rules apply to raw emails and attachments?
10. How should raw emails, profile pictures, comment images/GIFs, and other uploaded files be stored?
11. Which browser token storage and password-recovery email delivery will be used? The initial signing, hashing, rotation, and session implementation is documented in the API guide.
12. What security, privacy, logging, monitoring, backup, and operational requirements are needed before production?
13. What exactly do Unresolved and Resolved mean for assessments, including assessments with multiple tickets or no tickets?

## 19. Future extensions

- Additional mail providers and push/webhook ingestion.
- Notifications and escalation rules.
- Service-level reporting and trend analytics.
- Parser plug-ins/version migration tools.
- External ticketing or chat integrations.
