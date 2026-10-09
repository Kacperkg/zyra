# Zyra web

React frontend using Vite, TanStack Router, React Context and colocated CSS Modules. No Redux or Tailwind. See the [project overview and Docker demo](../README.md), [API setup](../zyra-api/README.md) and [product requirements](../documentation.md).

## Local development

Requires Node 22.13+ and npm. From `zyra-web/`, with the API running:

```sh
npm ci
npm run dev
```

Open the address printed by Vite, normally `http://127.0.0.1:5173`. Vite binds to loopback and proxies `/api` to `http://127.0.0.1:8080`. For the API README's port-8081 example:

```sh
API_PROXY_TARGET=http://127.0.0.1:8081 npm run dev
```

The frontend does not provide its own authentication or sample data. The API's opt-in local fixture supplies `admin` / `admin` with owner role; otherwise sign in using an account provisioned through the API. All roles share one login screen; there is no public registration.

The [root Docker demo](../README.md#local-demo) serves the built frontend on port 3000 through nginx, which proxies API requests and supports direct navigation/refresh on client routes. `npm run preview` is a build preview, not a replacement for that API proxy setup.

## Implemented screens

- Login and a restrained Oracle/SQL dashboard; SQL is visibly unavailable.
- Separate Open/Closed Oracle issue lists with URL search/filters/sorting and Modern/Classic appearance.
- Ticket detail with Discussion and Raw Email tabs, grouped system findings, coloured status badges, participants and five linked similar issues.
- Profile name, HTTPS avatar URL, light/dark and appearance preferences, plus private saved tickets with editable personal titles/search/pagination.
- Rich comments, author-only editing/deletion, direct GIF URLs, mentions and notification bell.

Client/database, assessment-history and settings/account-administration pages are not implemented. Client/database names stay text until those destinations exist. Password controls, general uploads and ticket-specific note editing remain unfinished. SQL functionality is Release 1.0 scope.

## Structure

```text
src/api/          Domain request modules and shared authenticated client
src/components/   Reusable UI, navigation, layout, editor and ticket components
src/contexts/     Authentication and account theme/appearance preferences
src/pages/        Page components with colocated .module.css
src/routes/       Thin TanStack file-based route definitions
src/styles/       Global tokens, reset and base styles
src/types/        API contracts and shared pure helpers
tests/            Node-based .mjs tests
```

Keep domain-specific components together within `components/` and reuse general controls where appropriate. API modules own network requests. Context holds session/preferences; page/component state holds local interactions. The Vite router plugin generates `src/routeTree.gen.ts`; do not hand-edit it. Global colour/radius tokens live in `src/styles/tokens.css`.

## Interaction contracts

### Issue lists and tickets

Fetch 50 summary rows on load, another 50 automatically on scroll, then use Next page after at most 100. Both appearances share API queries and URL state; filter changes reset paging. Do not preload comments, raw bodies or full findings. Incoming changes can shift offset pages; Refresh restarts the current view, and duplicate IDs are removed locally.

Modern is default. Classic spans most of the screen with small gutters and uses a compact striped seven-column grid; it omits redundant status because the route selects Open or Closed. Classic dates/times are browser-local `YYYY-MM-DD` / `HH:mm:ss`; Modern retains locale formatting and its status column. Narrow screens scroll within the table rather than overflowing the page.

Comment, Comment and close, separate Close and Reopen are independent actions. Comment and close requires content. Closed tickets accept further comments without reopening; Reopen is shown at the top and bottom. Author-only edit/delete icons stay at each comment's top-right. Edits preserve chronological position and show Edited; deletion requires confirmation. Status events remain immutable when a closure comment is edited/deleted. Database notes and ticket notes are labelled separately.

### Rich text, GIFs and mentions

The editor uses accessible formatting icons, alignment, text sizes/colour, lists, links, URL detection, GIF insertion and active-user name search. It submits the versioned JSON API contract, and the renderer uses escaped React content rather than arbitrary pasted HTML.

Direct HTTPS links with a `.gif` path become GIFs on paste or submission; explicit Insert link remains a link. Copied browser images work when clipboard HTML includes a direct HTTPS image address. Pixel-only images require the deferred upload feature. Tenor webpage URLs remain ordinary links: there is no API-key resolver, scraping, provider search or embed integration. Avatars and Add GIF require direct image addresses; failed images have a fallback.

The notification bell polls every 30 seconds while signed in. Opening the list does not mark notifications read; clicking one marks it read and navigates to its ticket. Mark all read is explicit. Historical mention identities remain readable after retirement.

### Session and preferences

Sessions persist in local storage across reloads/browser restarts until the fixed seven-day deadline. Access tokens last up to 15 minutes; rotation updates storage, and logout/invalid/expired sessions clear it. Concurrent refresh requests are coordinated, including across tabs with Web Locks where available. Storage events synchronize session updates/logout; stale requests cannot restore a replaced session. Blocked browser storage falls back to memory only.

Local storage is JavaScript-readable and does not protect tokens against XSS. The API remains authoritative for roles, retirement and session revocation; frontend visibility is not authorization.

Light and Modern are defaults. Theme/appearance are independent account preferences persisted through the API. Dark mode uses neutral surfaces; green remains the accent. Avatar URLs are loaded by the browser, not uploaded/proxied by the API; clearing the URL restores initials.

## Checks

```sh
npm test
npm run build
```

Tests use Node's test runner in `.mjs` files for session/refresh behaviour, safe content handling, query pagination and table formatting. They are not shipped to the browser. Build generates routes, creates production assets and checks TypeScript. Browser/API workflows require a running API/database and are not covered solely by these helper tests.

Manual smoke check: sign in, refresh, edit profile/preferences, save/retitle a ticket, check list scrolling, post formatted text/GIF/mention, edit/delete your comment and test Close/Reopen independently. Use a second active account to verify mention notifications and comment ownership. Account provisioning is API-only for now.
