# Zyra web

React + Vite + TanStack Router, React Context and CSS Modules. Use Node 22.13+ (or a current supported version) and npm.

```sh
npm install
npm run dev
npm run build
npm test
```

Vite proxies `/api` to `http://127.0.0.1:8080`. Override for a different API port:

```sh
API_PROXY_TARGET=http://127.0.0.1:8081 npm run dev
```

Implemented: shared login, dashboard, separate open/closed Oracle lists in Modern/Classic layouts, ticket detail with Discussion/Raw Email, grouped system findings, coloured status badges, rich comments, author-only edit/delete, GIF URLs, mentions, private saved tickets, a profile page and notification bell. Comment, Comment and close, separate Close and Reopen remain independent actions. Lists load 50 then another 50 on scroll, followed by Next page. Both layouts share the same API/search/filter/sort/paging logic; table headings also support sorting. Client/database choices load on demand in batches capped at 50. Results deduplicate by ID; incoming changes can shift offset pages, so Refresh restarts the current view.

Classic uses a compact Issues panel with refresh, search and shared filters, plus a striped seven-column grid: number, check type, created date/time, client, database and assessment type. Dates are browser-local `YYYY-MM-DD`, times `HH:mm:ss`; Modern keeps locale formatting and its status column. Open/closed selection stays in navigation, not the list. No page-size selector is added. Small screens use keyboard-accessible horizontal table scrolling. Client/database names remain text until those destination screens exist.

Authentication sessions are persisted in local storage, so refreshing or restarting the browser retains login until the fixed seven-day session deadline. Rotated tokens and profile updates are persisted; logout and expired/invalid stored sessions clear the record. Refresh requests are coordinated, including across tabs with Web Locks where supported, and cannot restore a logged-out or replaced session. Other tabs observe session changes. If browser storage is blocked, login falls back to memory only. Local storage is accessible to JavaScript and does not protect tokens from XSS. All roles use the same login. There is no registration or superadmin flow. The API can separately enable its development-only admin account.

Light and Modern are defaults. Theme/appearance preferences are saved to the account and restored on sign-in. The profile page at `/profile` accepts an HTTPS avatar URL (empty restores initials), allows name editing and lists private bookmarks with editable personal titles, search and pages of 50. Failed avatars fall back to initials. Pages/components use colocated CSS Modules and global tokens. File-based routes stay thin; the Vite router plugin generates `src/routeTree.gen.ts`. API modules own network calls; Context owns the authentication session and account display preferences. Shared Avatar, StatusBadge, editor, renderer and notification controls are reused.

The native rich editor offers bold/italic/underline/strike, lists, alignment, text sizes/colour, Insert link, automatic typed-URL detection, GIF URLs and an active-user mention picker. It sends the versioned JSON API contract; pasted/dropped HTML is reduced to text, and rendering uses escaped React nodes. GIFs load directly from HTTPS sources with failure text. General image/file uploads and GIF search providers are not included. Author edits retain chronological position and show Edited; deletion requires confirmation and retains any closure event. A closed ticket accepts more comments without reopening.

The bell polls every 30 seconds while signed in, cleans up when the session changes, and pages notification history. Opening the list does not mark it read. Individual notification clicks mark it read before navigating to its ticket; mark-all-read is explicit. Mentions use server-supplied display identities so historical mentions remain readable even when recipients retire.

Pending: client/database pages, assessments, settings/admin-user management, image uploads and separate ticket-note editing. Disabled navigation is labelled accordingly. SQL remains Release 1.0. Raw email is rendered as escaped text.

The editor uses accessible icon buttons for formatting. Pasting a direct HTTPS URL whose path ends in `.gif` inserts an inline GIF immediately, including Tenor media links; typing one converts it on submission. Surrounding text is preserved, normal webpage URLs remain links, and explicitly inserted links remain links. Tenor `/view/` page links are not resolved and remain ordinary comment links; Add GIF and avatar fields explain that a direct image address is required. Copied browser images can also be pasted when their clipboard HTML includes a direct HTTPS image address; external HTML is never inserted. Pixel-only clipboard images still require the deferred upload/storage feature and show an explanation. No provider API key or resolver is used.

Author-only comment edit/delete icons sit in each comment's top-right header, including comments posted with Comment and close. Closure/reopen history appears as compact activity lines rather than editable comment cards; changing or deleting the comment does not undo closure. Closed tickets show Reopen in both the header and bottom comment actions.

`npm test` exercises authentication/profile refresh concurrency, session boundaries and content helpers (safe URLs, autolinks, list grouping and meaningful comments). `npm run build` generates the route tree, builds production assets and checks TypeScript. Live API/browser checks require the API and database running.

For a manual check, sign in, open Account → Profile & saved tickets, save an avatar URL, switch layouts/themes, and sign out/in to confirm persistence. Open a ticket, save a personal bookmark title, post formatted text/a GIF/mention, edit/delete your own comment and test the independent close/reopen actions. Sign in as a second active user to verify mention notifications and that they cannot edit your comments. Administrators provision that second account through the API; there is no user-management screen yet.
