# RFC: Account lifecycle, profile preferences, saved tickets, rich comments and notifications

Status: Draft for review

Date: 2026-09-25

Scope: API changes supporting the next Zyra frontend iteration

Related: [Product requirements](documentation.md), [current API contract](zyra-api/doc/API.md), [PoC delivery RFC](RFC.md)

## 1. Summary and decision requested

Extend the existing API with a single owner role, active/retired account lifecycle, account appearance preferences, private ticket bookmarks, editable rich comments, user mentions and in-app notifications. Establish these contracts before building their frontend interfaces.

Product behaviour marked **Confirmed** reflects the agreed plan. Endpoint names, schemas and implementation mechanisms marked **Proposed** require technical review. **Open** items must be resolved before their dependent work. This RFC does not claim these features already exist and does not replace the broader PoC RFC.

Review the proposed persistence boundaries, authorization rules, comment migration and notification lifecycle. Select an editor-compatible JSON schema before implementing rich-comment storage.

## 2. Problem and current baseline

Operators need clearer discussions, the ability to draw another person's attention to a ticket, and a private way to retain useful fixes. Users also need profile images and a choice between Modern and Classic presentation.

The API currently stores profile name, email, role and Light/Dark theme. It supports plain-text comments, comment-and-close, independent close/reopen actions and paginated ticket events. Comment text currently lives directly on `TicketEvent`, including `comment_and_close` events. It has no bookmark, mention, notification, avatar-URL or Modern/Classic persistence.

The initial web application uses an in-memory authentication session. This RFC preserves token storage and session lifetimes, adds `owner` alongside normal/trusted/admin, and replaces the current `disabled` flag with `status: active | retired`. These role/status changes are planned, not yet implemented. The current development `admin/admin` fixture still has the admin role until this migration is implemented.

## 3. Goals and exclusions

**Confirmed goals:**

- Exactly one owner, with retirement permissions below that role and archive-like account retirement/reactivation.
- Account-level HTTPS profile-image URL and persisted appearance preferences.
- Private saved tickets with optional personal titles.
- Validated structured rich comments with a plain-text representation.
- Author-only editing/deletion, preserving original comment order.
- Name-based active-user lookup for a mention picker.
- In-app mention notifications with unread state and ticket links.
- Compatibility with existing comments and independent ticket-status history.

**Outside this change:**

- Images/GIFs in comments, file uploads and media storage.
- Building the profile screen, editor, notification bell or Classic issues layout.
- Redesigning the system findings presentation; system findings remain non-editable.
- Public registration, a separate superadmin role, ownership transfer or administrative overrides of comment ownership. The owner role is included in this change.
- Email/push notifications, live delivery infrastructure, automatic ticket merging/closure.
- Changing the agreed issue-list pagination or interpreting assessment resolution states.

## 4. Confirmed product rules

### 4.0 Account roles and lifecycle

- Roles are `owner`, `admin`, `trusted` and `normal`. There is exactly one owner, who cannot be retired or demoted. Ownership transfer is not included.
- The existing local `admin/admin` test account becomes the owner during implementation; its login credentials remain unchanged and development-only restrictions still apply. This is an explicit fixture choice, not a first-user-becomes-owner rule.
- Every account has one status: `active` or `retired`. New accounts default to `active`. Do not keep an overlapping `disabled` control.
- Active describes account eligibility, not online presence. An active user may log in and participate whether or not they are currently signed in.
- Retirement archives the account: deny login, access-token authentication, refresh and all participation, and revoke every existing session/refresh token. Previously issued JWTs must be denied by the existing server-side session/account checks.
- Preserve the retired person's identity, role, profile, comments, attribution, bookmarks and existing notifications. Retirement does not delete their history or withdraw existing notifications merely because the recipient retired.
- Reactivation restores the same account and retained data. A fresh login is required; revoked sessions never become valid again.
- Admins can retire trusted and normal users, but cannot retire themselves, other admins or the owner. The owner can retire non-owner accounts, including admins.

**Proposed application of the same boundary to reactivation:** admins may reactivate retired trusted/normal accounts; only the owner may reactivate retired admins. The owner inherits ordinary admin capabilities but gains no override for private bookmarks, notifications or author-only comment editing/deletion.

**Proposed role-assignment boundary:** only the owner grants/removes the admin role; admins manage trusted/normal accounts only. Reject combined role/status updates that would demote another admin first to bypass retirement restrictions. Role-assignment details still require review; the retirement restrictions above are confirmed.

### 4.1 Profile and appearance

- All active users may update their own profile-image URL. Only HTTPS URLs are accepted; clearing the value restores an initials fallback.
- The browser loads the image directly. Zyra stores the URL and does not fetch, proxy or upload the image.
- Modern is the default presentation. Classic is an account preference intended initially for the issues-page presentation.
- Modern/Classic and Light/Dark are independent preferences, persisted to the account and applied across devices.
- A failed image load must not prevent using the profile or application.

### 4.2 Private saved tickets

- Each user can save or unsave tickets for themselves, at most once per ticket.
- An optional personal title can be added or edited later. Without one, display the ticket's title.
- Saving does not change ticket status or expose a bookmark to other users. Admin role grants no access to another user's bookmarks.
- List bookmarks with search and pagination, newest saved first. Editing the title does not move a bookmark to the top.

### 4.3 Rich comments and ownership

- Support text formatting and links; images/GIFs are deferred. The first text scope covers paragraphs, bold/italic/underline, lists, alignment, text sizes/colours and links. The exact allowed values belong in the selected content schema.
- Store structured JSON and derive plain text on the server for previews/search. Do not trust independently supplied plain text or mention lists.
- Only the author may edit or delete their comment. Admins cannot edit/delete another author's comment.
- Edited comments display an Edited marker. Keep their original creation timestamp and chronological position.
- Deleted comments disappear from the visible timeline without a deletion placeholder. Internal retention is a separate open decision; this does not authorize exposing deleted text to admins.
- System findings and ticket-status events are non-editable and non-deletable through comment endpoints.
- Editing/deleting the comment accompanying a closure never reverses or erases that closure. Comments on closed tickets remain allowed without reopening.
- Comment and close still requires meaningful, non-empty comment content and remains atomic.

### 4.4 Mentions and notifications

- Active accounts of any role may mention other active accounts, including users who are offline or signed out. The frontend offers name-based suggestions while typing; lookup results must be bounded. Retired users do not appear in suggestions.
- Persist mentions using user IDs, so a renamed user remains the same recipient. Historical mentions of someone who later retires remain attributed to them; retirement does not rewrite existing comments.
- Notify only after submission, once per recipient per comment. Repeated mentions in one comment are deduplicated. Do not notify the author about self-mentions.
- On edit, newly added recipients are notified only if they have never been notified for that comment. Removing and re-adding the same person does not notify them again.
- Removing a mention withdraws that person's notification. Deleting a comment withdraws all of its notifications, including read notifications.
- Other read notifications remain in the recipient's history, subject to a future retention policy.
- Opening a notification marks it read and navigates to its ticket. Provide individual and mark-all-read actions. Merely opening the bell does not mark everything read.
- The initial frontend polls every 30 seconds while open. Live delivery is a possible later addition.

## 5. Proposed API contracts

All paths below are relative to `/api`, require the existing authenticated session, and use the existing error conventions. Resource ownership is derived from the authenticated user, never accepted as an editable request field.

| Method and path | Proposed behaviour |
| --- | --- |
| `GET /admin/users?status=...` | Owner/admin user listing with explicit active/retired filtering so archived accounts can be managed. Include role/status, not credential material. |
| `POST /admin/users` | Owner/admin account creation; default status to active. Restrict assignable roles, and reject creation of another owner. |
| `PATCH /admin/users/:id` | Replace `disabled` input with validated `status`; enforce actor/target role boundaries and atomic session revocation on retirement. Reactivation never restores sessions. |
| `GET /users/me` | Include `avatar_url` and `appearance` alongside existing profile fields. |
| `PATCH /users/me` | Update own `avatar_url`, `appearance` (`modern`/`classic`) and existing `theme`; preserve existing profile restrictions. |
| `GET /users/mention-options?q=...&limit=...` | Bounded active-user name search returning minimal identity fields. Not the admin user-management response. |
| `GET /users/me/saved-tickets` | Paginated private bookmark summaries; search personal title and ticket display information. |
| `PUT /users/me/saved-tickets/:ticket_id` | Idempotently save a ticket or update its optional personal title. Preserve initial save time on updates. |
| `DELETE /users/me/saved-tickets/:ticket_id` | Idempotently remove the caller's bookmark. |
| `POST /tickets/:id/comments` | Existing route accepts the agreed rich-content envelope; transition compatibility described below. |
| `POST /tickets/:id/comment-and-close` | Same content contract, with atomic comment creation and closure. |
| `PATCH /tickets/:id/comments/:comment_id` | Author-only content replacement, revision check, mention reconciliation and Edited timestamp. |
| `DELETE /tickets/:id/comments/:comment_id` | Author-only removal from visible discussions and notification withdrawal; retain independent status history. |
| `GET /users/me/notifications` | Paginated recipient-only notifications with optional unread filter. |
| `GET /users/me/notifications/unread-count` | Count active unread notifications for the bell. |
| `PATCH /users/me/notifications/:id/read` | Idempotently mark an owned notification read. |
| `POST /users/me/notifications/read-all` | Mark current active notifications read without touching other recipients. |

Ticket detail should include caller-specific bookmark state and title, avoiding one additional lookup per visible ticket. Bookmark lists contain compact ticket summaries, not raw reports or full discussions.

Notification responses should contain notification ID, ticket ID/number/title, comment ID, actor display identity, creation time and read time. Linking to a particular comment is useful, but locating that comment within a paginated timeline is a later frontend contract; a valid ticket link is required now.

**Proposed limits:** use the existing 50-item list default and validated maximum for saved tickets/notifications; cap mention results at 20 with at least two typed characters. Return no unbounded directory results. Exact text/URL limits and supported link protocols must be agreed with the editor contract.

## 6. Proposed persistence and transaction design

### 6.1 Accounts, profiles and bookmarks

Add a non-null status with an `active` default and a database constraint permitting only `active` and `retired`. Enforce that the owner is active. A partial unique constraint/index can enforce at most one owner; explicit bootstrap/migration and refusal to delete/demote/retire that owner preserve exactly one. Do not assume uniqueness alone guarantees that an owner exists.

Retirement should lock and re-read the target and relevant actor state, check permissions, change status and revoke sessions in one transaction. Authentication/refresh must consult current status. Authorization for mutations must recheck the actor so an in-flight request cannot commit participation after retirement has taken effect. Serialize affected account mutations consistently to avoid races between retirement, reactivation and role changes.

Record actor, target, prior/new status and timestamp in an account-lifecycle audit record. This records administrative transitions without changing ticket history. Notification creation must validate current recipient eligibility; proposed handling of stale/new mentions is listed in the decision register.

Add nullable `avatar_url` and `appearance` with a `modern` default to users. Preserve existing theme values. Validate HTTPS URL structure, reject embedded credentials and unsupported schemes, and avoid server-side remote-image requests. A stored valid URL is not a promise that an image is available.

Introduce `saved_tickets` with user ID, ticket ID, optional title and saved timestamp. A unique `(user_id, ticket_id)` constraint enforces private bookmark identity. Use saved timestamp plus a deterministic ID tie-breaker for paging. Verify ticket access before creating or resolving a bookmark.

### 6.2 Comments versus timeline events

Introduce a separately addressable comment record containing author, ticket, schema version, JSON content, derived plain text, original creation time, edited time and revision. Link the existing timeline event to that comment. Keep ticket transitions as independent immutable events.

This separation is needed because deleting the editable text of an existing `comment_and_close` event must not delete the closure evidence. The timeline may still show the retained closure event; it must not show a comment-deleted placeholder or expose the deleted text.

Use optimistic revisions for edits to reject stale writes rather than silently overwriting a concurrent edit. Lock/recheck ownership and deletion state within the mutation transaction. Deleting a comment must not change its ticket's state.

The API validates document nodes, marks, nesting, attributes and size/depth limits using an allowlist. Extract text and mention IDs from the validated document. Link destinations must be validated; raw HTML, script URLs, arbitrary styles and executable embeds are excluded. Plain-text serialization must preserve meaningful separation between paragraphs and list items.

### 6.3 Mention identity and notification lifecycle

Maintain current comment mentions and a persistent delivery ledger keyed by `(comment_id, recipient_id)`. Retain the ledger after a mention is removed so removing/re-adding it cannot generate another notification.

Store recipient, actor, ticket, comment, created time, read time and withdrawal state on notifications. Withdrawal excludes the notification from lists and unread counts. A removed/re-added mention stays withdrawn under the once-ever rule; it does not reappear as unread or regain a new creation time.

Create/update/delete comment content, current mention associations, delivery-ledger entries and notifications in one transaction. For comment-and-close, include the ticket transition and its immutable event in that same transaction. Uniqueness constraints prevent duplicate recipient notifications during concurrent submissions or retries against the same comment identity.

Posting a completely new comment is distinct from retrying an existing submission. A request-id/idempotency strategy for comment creation is **Open**; do not claim that notification uniqueness alone prevents duplicate comments after an ambiguous network failure.

Recipient-owned read operations should be idempotent and concurrency-safe. Proposed mark-all-read uses a server-side cutoff captured at request start so later notifications remain unread. A stale client following a withdrawn notification should receive an unavailable result and refresh its list/count.

## 7. Migration and compatibility

Before the content migration, backfill account status from the old flag: `disabled=false` becomes active and `disabled=true` becomes retired. Revoke sessions belonging to retired accounts. Switch all authentication, authorization, administrative payloads and mention queries to status, then remove `disabled`; do not support two independently editable lifecycle controls. Document the API contract change and reject obsolete `disabled` writes explicitly.

Promote the explicitly identified development `admin/admin` fixture to owner. For any non-demo installation, require explicit owner selection during migration/bootstrap; do not promote the first account or all admins. Preserve user IDs and associated history. Update role validators, admin middleware and service permissions to recognize owner consistently, without bypassing resource ownership. Do not silently promote a newly created demo fixture over an existing owner.

1. Add profile fields with defaults preserving current users and theme choices. Add bookmark/comment/mention/notification structures with reviewed constraints and indexes.
2. Backfill existing user-comment text into a versioned plain-paragraph document, preserving author, ticket and original timestamp. Do not backfill system findings as user-editable comments.
3. Split the editable comment portion of historical `comment_and_close` records from their immutable closure semantics without losing closure reporting or changing historical order. Define stable IDs/mapping before migration.
4. Temporarily accept the current `{comment: "..."}` request shape by converting it server-side. Introduce an explicit schema-versioned content envelope; reject ambiguous requests carrying conflicting old and new bodies.
5. Extend timeline responses with comment identity, content, revision and edited metadata while maintaining existing text rendering during frontend transition. Do not invent an Edited date for migrated comments.
6. Preserve existing participant and admin closure-history behaviour. Decide separately how deleted comments affect participant membership; status actors must remain attributable.

Production migration tooling remains governed by the wider project decision. Development AutoMigrate does not replace the required content backfill, compatibility tests or rollback planning.

## 8. Alternatives and tradeoffs

| Area | Proposed choice and reason | Alternative/tradeoff |
| --- | --- | --- |
| Avatar | Store HTTPS URL and let the browser load it; matches the requested URL-only profile. | Uploads/proxying introduce media storage and server fetch responsibilities, outside this stage. Remote images may fail and reveal requests to their host. |
| Comment body | Validated, versioned JSON plus derived text supports formatting and user-ID mentions. | HTML requires a different sanitization contract; plain text cannot retain the requested formatting. JSON still requires validation and safe rendering. |
| Editable text | Separate comments from immutable ticket events. | Mutating/deleting combined closure events risks losing operational history. Separation requires migration work. |
| Notification delivery | Durable database records with 30-second polling. | Live delivery reduces latency but adds connection/reconnection and multi-instance delivery concerns. Keep it independent of storage/read APIs. |
| Deletion | Hide the comment from the discussion and withdraw notifications. | Internal soft deletion versus hard deletion remains open; neither implies a user-visible deletion placeholder. |

No editor, cache, broker or live-delivery library is selected by this RFC. Existing Go/Gin/GORM/PostgreSQL and React conventions continue to apply.

## 9. Risks and validation

- **Ownership bypass:** test that normal, trusted, admin and owner users cannot edit/delete others' comments, enumerate others' bookmarks, or read/mark others' notifications.
- **Lifecycle bypass:** test active defaults, all actor/target retirement combinations, owner immutability, single-owner enforcement, forged role/status fields and combined-update privilege bypasses. Verify retirement denies existing access/refresh tokens and fresh login; reactivation requires fresh credentials and preserves history. Include concurrent retirement versus refresh and participation tests.
- **Unsafe rich content:** test invalid document versions/nodes, deep or oversized content, unsafe links, forged mention IDs and attempts to edit system/status events.
- **Notification inconsistency:** test duplicate mentions, self-mentions, edits adding/removing recipients, remove/re-add, comment deletion, read withdrawal and concurrent writes.
- **History loss:** verify migrated comment-and-close records still support closure reporting, unchanged ticket status and stable event ordering after comment edits/deletion.
- **Privacy:** return minimal mention-picker data; never expose password/session fields. Decide how duplicate display names are disambiguated before finalizing the response.
- **Scale:** use bounded search and lists, indexed recipient/read/withdrawal queries and bookmark uniqueness. Benchmark representative counts; polling should stop after sign-out and should not stack overlapping requests.
- **Concurrent user changes:** retired users disappear from suggestions; online presence does not affect results. Validate recipients again on submission. The handling of stale retired-user mentions remains an explicit decision below.

## 10. Delivery sequence and acceptance

### Stage 0 — Owner and account lifecycle

Implement the status migration, explicit owner selection, role-aware administration, retirement/session revocation and reactivation before dependent mention work. Accept when admins can retire only trusted/normal users, the owner can manage non-owner retirement, nobody can retire/demote the owner or create another one, retired users cannot authenticate/participate, and reactivation restores the retained account only through a fresh login. Confirm the proposed role-assignment/reactivation boundaries before implementing them.

### Stage A — Profile and saved tickets

Implement profile validation/defaults, private bookmark ownership, unique save semantics, title updates, search and pagination. Accept when preferences survive a fresh login, invalid avatar URLs are rejected, clearing works, bookmarks stay private and retitling does not change saved order.

### Stage B — Comment contract and migration

Select the editor/schema; define supported formatting/links and limits. Implement JSON validation, derived text, compatibility/backfill, author-only edit/delete and stable timeline ordering. Accept when old and new comments render through the API, only authors can mutate their comments, Edited metadata is correct, deleted comments disappear, and closure history remains intact.

### Stage C — Mentions and notifications

Implement active-user search, validated mentions, transactional notification creation/reconciliation, recipient-only lists/counts and read actions. Accept when all agreed mention lifecycle cases pass and the API supports a 30-second polling frontend without duplicate notifications.

Update the API documentation and regression tests with each stage. The later frontend work consumes these contracts for the editor, profile, bookmarks and bell; the Classic layout remains its own visual implementation stage.

## 11. Open decisions before dependent implementation

1. Editor-compatible JSON format, versioning and exact allowed formatting/link attributes. Agree size limits and what constitutes meaningful non-empty content.
2. Internal deletion retention and audit policy. Deleted text is not visible in the ordinary timeline; administrator access to prior revisions has not been approved.
3. Stable mapping/backfill for historical comment-and-close events, including participant behaviour after deletion.
4. Mention-picker disambiguation fields and handling when a selected recipient retires before submission. Proposed: reject newly introduced retired-user mentions while allowing an edit to preserve a pre-existing historical mention without new notification delivery.
5. Comment creation idempotency for ambiguous retries; edits use proposed revisions and notifications use recipient/comment uniqueness.
6. Notification retention and any future live-delivery mechanism. Neither blocks the initial durable polling API once the initial retention policy is explicit.
7. Confirm proposed role-assignment and reactivation permissions: only owner grants/removes admin and reactivates admins; admins manage trusted/normal accounts. Retirement limits, owner immutability and the development account's owner identity are already confirmed.

Review does not authorize a frontend editor library or new infrastructure by implication. These decisions should be recorded before their respective implementation stage begins.
