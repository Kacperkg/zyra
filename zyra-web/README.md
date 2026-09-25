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

Implemented: shared login, dashboard, separate open/closed Oracle lists, ticket detail with discussion/raw email, comments/close/reopen, similar issues, navbar and themes. Lists load 50 then another 50 on scroll, followed by a Next page action. URL filters reset paging. Client/database choices load on demand in batches capped at 50, with server search for further choices. Results deduplicate by ID; incoming changes can shift offset pages, so Refresh restarts the current view.

Authentication tokens remain in memory; refreshing the browser requires sign-in. Refresh requests are serialized and cannot restore a logged-out or replaced session. All roles use the same login. There is no registration or superadmin flow. The API can separately enable its development-only admin account.

Light is default; theme preference currently resets on page reload. Pages and components use colocated CSS Modules and global tokens. File-based route definitions stay thin; the Vite router plugin generates `src/routeTree.gen.ts`. API modules own network calls; Context owns authentication/theme only.

Pending: clients/database pages, assessments, settings/profile, rich text/media, separate ticket notes, and persistent token storage. Disabled navigation is labeled accordingly. SQL remains Release 1.0. Raw email is rendered as escaped text.

`npm test` exercises authentication concurrency and session boundaries using Node's test runner. `npm run build` generates the route tree, builds production assets and checks TypeScript. Live API/browser checks require the API and database running.
