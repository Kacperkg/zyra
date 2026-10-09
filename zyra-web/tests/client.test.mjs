import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { stripTypeScriptTypes } from 'node:module';

const source = stripTypeScriptTypes(await readFile(new URL('../src/api/client.ts', import.meta.url), 'utf8'));
let moduleID = 0;
const load = () => import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}#${moduleID++}`);
const session = (token = 'original') => ({access_token: token, refresh_token: `refresh-${token}`, access_expires_at: new Date(Date.now() + 60000).toISOString(), session_expires_at: new Date(Date.now() + 600000).toISOString(), user: {id: token, name: token, email: `${token}@example.test`, role: 'normal'}});
const json = (body, status = 200) => new Response(JSON.stringify(body), {status, headers: {'Content-Type': 'application/json'}});
const deferred = () => { let resolve; const promise = new Promise(r => { resolve = r; }); return {promise, resolve}; };

test('concurrent unauthorized requests share a single refresh', async t => {
  const client = await load();
  client.setSession(session());
  const gate = deferred();
  let refreshCount = 0;
  t.mock.method(globalThis, 'fetch', async (url, options) => {
    if (url.endsWith('/auth/refresh')) { refreshCount++; await gate.promise; return json(session('rotated')); }
    return options.headers.Authorization === 'Bearer original' ? json({error: 'expired'}, 401) : json({ok: true});
  });
  const first = client.request('/one');
  const second = client.request('/two');
  await new Promise(r => setImmediate(r));
  gate.resolve();
  assert.deepEqual(await Promise.all([first, second]), [{ok: true}, {ok: true}]);
  assert.equal(refreshCount, 1);
});

test('logout clears in-memory session even when the API is unreachable', async t => {
  const client = await load();
  client.setSession(session());
  t.mock.method(globalThis, 'fetch', async () => { throw new Error('offline'); });
  await assert.rejects(client.logout(), /offline/);
  await assert.rejects(client.request('/tickets'), /sign in/i);
});

test('logout during refresh neither restores the session nor sends an unauthorized retry', async t => {
  const client = await load();
  client.setSession({...session(), access_expires_at: new Date(0).toISOString()});
  const gate = deferred();
  const requests = [];
  t.mock.method(globalThis, 'fetch', async (url) => {
    requests.push(url);
    if (url.endsWith('/auth/refresh')) { await gate.promise; return json(session('rotated')); }
    return json({private: true});
  });
  const pending = client.request('/tickets');
  client.setSession(null);
  gate.resolve();
  await assert.rejects(pending);
  assert.deepEqual(requests, ['/api/auth/refresh']);
  await assert.rejects(client.request('/tickets'), /sign in/i);
});

test('an old request cannot retry under a newly signed-in account', async t => {
  const client = await load();
  client.setSession(session('first-user'));
  const gate = deferred();
  const authorizations = [];
  t.mock.method(globalThis, 'fetch', async (_url, options) => {
    authorizations.push(options.headers.Authorization);
    if (authorizations.length === 1) { await gate.promise; return json({error: 'expired'}, 401); }
    return json({private: 'second-user'});
  });
  const pending = client.request('/tickets');
  client.setSession(session('second-user'));
  gate.resolve();
  await assert.rejects(pending);
  assert.deepEqual(authorizations, ['Bearer first-user']);
});

test('a forbidden response does not rotate credentials', async t => {
  const client = await load();
  client.setSession(session());
  const requests = [];
  t.mock.method(globalThis, 'fetch', async url => { requests.push(url); return json({error: 'Forbidden'}, 403); });
  await assert.rejects(client.request('/admin/users'), /Forbidden/);
  assert.deepEqual(requests, ['/api/admin/users']);
});

test('an expired seven-day session fails before requesting refresh', async t => {
  const client = await load();
  client.setSession({...session(), access_expires_at: new Date(0).toISOString(), session_expires_at: new Date(0).toISOString()});
  const mock = t.mock.method(globalThis, 'fetch', async () => json({}));
  await assert.rejects(client.request('/tickets'), /Session expired/);
  assert.equal(mock.mock.callCount(), 0);
});

test('successful stale responses are rejected after a session switch', async t => {
  const client = await load();
  client.setSession(session('first-user'));
  const gate = deferred();
  t.mock.method(globalThis, 'fetch', async () => { await gate.promise; return json({private: 'first-user'}); });
  const pending = client.request('/tickets');
  client.setSession(session('second-user'));
  gate.resolve();
  await assert.rejects(pending, /Session changed/);
});

test('profile update preserves in-flight requests and rotated credentials', async t => {
  const client = await load();
  const current = {...session(), access_expires_at: new Date(0).toISOString()};
  client.setSession(current);
  let observed;
  client.watchSession(value => { observed = value; });
  const gate = deferred();
  t.mock.method(globalThis, 'fetch', async url => {
    if (url.endsWith('/auth/refresh')) { await gate.promise; return json({...session('rotated'), user: {...current.user, role: 'trusted'}}); }
    return json({ok: true});
  });
  const pending = client.request('/tickets');
  client.updateSessionUser({...current.user, name: 'Updated name', theme: 'dark'});
  gate.resolve();
  assert.deepEqual(await pending, {ok: true});
  assert.equal(observed.access_token, 'rotated');
  assert.equal(observed.user.name, 'Updated name');
  assert.equal(observed.user.theme, 'dark');
  assert.equal(observed.user.role, 'trusted');
});

test('partial profile updates cannot overwrite a newer independent preference', async () => {
  const client = await load();
  const current = session();
  client.setSession(current);
  let observed;
  client.watchSession(value => { observed = value; });
  client.updateSessionUser({...current.user, theme: 'dark'}, ['theme']);
  client.updateSessionUser({...current.user, theme: 'light', name: 'New name'}, ['name']);
  assert.equal(observed.user.theme, 'dark');
  assert.equal(observed.user.name, 'New name');
  client.updateSessionUser({...session('another-user').user, name:'Wrong'}, ['name']);
  assert.equal(observed.user.name, 'New name');
});
