import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
const source = stripTypeScriptTypes(
  await readFile(new URL("../src/api/client.ts", import.meta.url), "utf8"),
);
let id = 0;
const load = () =>
  import(
    `data:text/javascript;base64,${Buffer.from(source).toString("base64")}#storage-${id++}`
  );
const key = "zyra.session.v1";
const fixture = () => ({
  access_token: "access",
  refresh_token: "refresh",
  access_expires_at: new Date(Date.now() + 60000).toISOString(),
  session_expires_at: new Date(Date.now() + 604800000).toISOString(),
  user: {
    id: "user",
    name: "User",
    email: "user@example.test",
    role: "normal",
    status: "active",
  },
});
const json = (value) =>
  new Response(JSON.stringify(value), {
    headers: { "Content-Type": "application/json" },
  });
function storage(t) {
  const values = new Map();
  const previous = Object.getOwnPropertyDescriptor(globalThis, "localStorage");
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: {
      getItem: (k) => values.get(k) ?? null,
      setItem: (k, v) => values.set(k, v),
      removeItem: (k) => values.delete(k),
    },
  });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, "localStorage", previous);
    else delete globalThis.localStorage;
  });
  return values;
}
test("login survives a new client instance; logout clears stored credentials", async (t) => {
  const values = storage(t);
  const client = await load();
  t.mock.method(globalThis, "fetch", async (url) =>
    json(url.endsWith("/login") ? fixture() : {}),
  );
  await client.login("user", "password");
  assert.equal(JSON.parse(values.get(key)).refresh_token, "refresh");
  const restored = await load();
  assert.equal(restored.getSession().user.id, "user");
  await restored.logout();
  assert.equal(values.has(key), false);
  assert.equal((await load()).getSession(), null);
});
test("malformed, retired and expired stored sessions are removed on startup", async (t) => {
  const values = storage(t);
  for (const value of [
    "{broken",
    JSON.stringify({
      ...fixture(),
      session_expires_at: new Date(0).toISOString(),
    }),
    JSON.stringify({
      ...fixture(),
      user: { ...fixture().user, status: "retired" },
    }),
    JSON.stringify({ ...fixture(), access_expires_at: "invalid" }),
  ]) {
    values.set(key, value);
    assert.equal((await load()).getSession(), null);
    assert.equal(values.has(key), false);
  }
});
test("expired access refreshes after reload and persists rotated credentials", async (t) => {
  const values = storage(t);
  const original = {
    ...fixture(),
    access_expires_at: new Date(0).toISOString(),
  };
  values.set(key, JSON.stringify(original));
  const client = await load();
  let rotations = 0;
  t.mock.method(globalThis, "fetch", async (url) => {
    if (url.endsWith("/refresh")) {
      rotations++;
      return json({
        ...original,
        access_token: "rotated",
        refresh_token: "rotated-refresh",
        access_expires_at: new Date(Date.now() + 900000).toISOString(),
      });
    }
    return json({ ok: true });
  });
  assert.deepEqual(await client.request("/tickets"), { ok: true });
  assert.equal(rotations, 1);
  assert.equal(JSON.parse(values.get(key)).refresh_token, "rotated-refresh");
  assert.equal((await load()).getSession().access_token, "rotated");
});
test("profile edits persist without replacing tokens", async (t) => {
  const values = storage(t);
  const client = await load();
  client.setSession(fixture());
  client.updateSessionUser({ ...fixture().user, name: "Updated" }, ["name"]);
  assert.equal(JSON.parse(values.get(key)).user.name, "Updated");
  assert.equal(JSON.parse(values.get(key)).refresh_token, "refresh");
});
test("two client instances serialize refresh rotation across tabs", async (t) => {
  const values = storage(t);
  values.set(
    key,
    JSON.stringify({
      ...fixture(),
      access_expires_at: new Date(0).toISOString(),
    }),
  );
  const first = await load(),
    second = await load();
  let rotations = 0;
  let queue = Promise.resolve();
  const previous = Object.getOwnPropertyDescriptor(navigator, "locks");
  Object.defineProperty(navigator, "locks", {
    configurable: true,
    value: {
      request: (_name, work) => {
        const pending = queue.then(work);
        queue = pending.catch(() => {});
        return pending;
      },
    },
  });
  t.after(() => {
    if (previous) Object.defineProperty(navigator, "locks", previous);
    else delete navigator.locks;
  });
  t.mock.method(globalThis, "fetch", async (url) => {
    if (url.endsWith("/refresh")) {
      rotations++;
      await new Promise((r) => setImmediate(r));
      return json({
        ...fixture(),
        session_expires_at: JSON.parse(values.get(key)).session_expires_at,
        access_token: "rotated",
        refresh_token: "rotated-refresh",
      });
    }
    return json({ ok: true });
  });
  await Promise.all([first.request("/one"), second.request("/two")]);
  assert.equal(rotations, 1);
  assert.equal(second.getSession().access_token, "rotated");
});
