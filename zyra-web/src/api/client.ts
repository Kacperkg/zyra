import type { Session, User } from "../types/api";
const storageKey = "zyra.session.v1";
function readStoredSession(): Session | null {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return null;
    const value = JSON.parse(raw);
    if (
      !value ||
      typeof value.access_token !== "string" ||
      !value.access_token ||
      typeof value.refresh_token !== "string" ||
      !value.refresh_token ||
      typeof value.access_expires_at !== "string" ||
      !Number.isFinite(Date.parse(value.access_expires_at)) ||
      typeof value.session_expires_at !== "string" ||
      !Number.isFinite(Date.parse(value.session_expires_at)) ||
      Date.parse(value.session_expires_at) <= Date.now() ||
      !value.user ||
      typeof value.user.id !== "string" ||
      !value.user.id ||
      typeof value.user.name !== "string" ||
      typeof value.user.email !== "string" ||
      !["owner", "admin", "trusted", "normal"].includes(value.user.role) ||
      value.user.status === "retired"
    ) {
      localStorage.removeItem(storageKey);
      return null;
    }
    return value as Session;
  } catch {
    try {
      localStorage.removeItem(storageKey);
    } catch {
      /* Storage may be blocked. */
    }
    return null;
  }
}
function persistSession() {
  try {
    if (session) localStorage.setItem(storageKey, JSON.stringify(session));
    else localStorage.removeItem(storageKey);
  } catch {
    /* Fall back to an in-memory session when browser storage is unavailable. */
  }
}
let session: Session | null = readStoredSession();
let onChange: (value: Session | null) => void = () => {};
let refreshing: Promise<void> | null = null;
let revision = 0;
export function getSession() {
  return session;
}
function sameSession(a: Session | null, b: Session | null) {
  return (
    !!a &&
    !!b &&
    a.user.id === b.user.id &&
    a.session_expires_at === b.session_expires_at
  );
}
// Synchronize token rotation and logout without writing back to other tabs.
if (typeof window !== "undefined")
  window.addEventListener("storage", (event) => {
    if (
      event.storageArea !== localStorage ||
      (event.key !== storageKey && event.key !== null)
    )
      return;
    const next = readStoredSession();
    if (!sameSession(session, next)) {
      revision++;
      refreshing = null;
    }
    session = next;
    onChange(session);
  });
export function watchSession(listener: typeof onChange) {
  onChange = listener;
  listener(session);
  return () => {
    onChange = () => {};
  };
}
export function setSession(value: Session | null) {
  revision++;
  refreshing = null;
  session = value;
  persistSession();
  onChange(value);
}
// Profile edits do not change the authentication generation or interrupt refresh.
export function updateSessionUser(user: User, fields?: (keyof User)[]) {
  if (!session || session.user.id !== user.id) return;
  const nextUser = fields
    ? {
        ...session.user,
        ...Object.fromEntries(fields.map((field) => [field, user[field]])),
      }
    : user;
  session = { ...session, user: nextUser };
  persistSession();
  onChange(session);
}
function assertSession(generation: number) {
  if (!session || revision !== generation)
    throw new Error("Session changed. Please sign in again.");
  return session;
}
async function failure(response: Response) {
  const body = await response.json().catch(() => ({}));
  return new Error(body.error || `Request failed (${response.status})`);
}
async function refresh() {
  if (refreshing) return refreshing;
  const current = session,
    generation = revision;
  if (!current || Date.parse(current.session_expires_at) <= Date.now()) {
    setSession(null);
    throw new Error("Session expired. Please sign in again.");
  }
  const rotate = async () => {
    assertSession(generation);
    // Another tab may already have rotated while this tab waited for the lock.
    const stored = readStoredSession();
    if (
      stored &&
      sameSession(current, stored) &&
      stored.refresh_token !== current.refresh_token
    ) {
      session = stored;
      onChange(session);
      return;
    }
    if (session!.refresh_token !== current.refresh_token) return;
    const response = await fetch("/api/auth/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: current.refresh_token }),
    });
    if (!response.ok) {
      if (revision === generation) setSession(null);
      throw await failure(response);
    }
    const next: Session = await response.json();
    assertSession(generation);
    const latest = assertSession(generation);
    const changedProfile = Object.fromEntries(
      (["name", "email", "avatar_url", "theme", "appearance"] as const)
        .filter((field) => latest.user[field] !== current.user[field])
        .map((field) => [field, latest.user[field]]),
    );
    session = {
      ...next,
      user: { ...next.user, ...changedProfile },
    };
    persistSession();
    onChange(session);
  };
  // Web Locks serialize refresh-token rotation across tabs of the same origin.
  refreshing = (async () => {
    if (typeof navigator !== "undefined" && navigator.locks)
      await navigator.locks.request("zyra.session.refresh", rotate);
    else await rotate();
  })().finally(() => {
    if (revision === generation) refreshing = null;
  });
  return refreshing;
}
export async function request<T>(
  path: string,
  options: RequestInit = {},
  raw = false,
): Promise<T> {
  if (!session) throw new Error("Please sign in.");
  if (Date.parse(session.session_expires_at) <= Date.now()) {
    setSession(null);
    throw new Error("Session expired. Please sign in again.");
  }
  const generation = revision;
  if (Date.parse(session.access_expires_at) <= Date.now() + 5000)
    await refresh();
  const usedToken = assertSession(generation).access_token;
  const send = () =>
    fetch(`/api${path}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options.headers,
        Authorization: `Bearer ${assertSession(generation).access_token}`,
      },
    });
  let response = await send();
  assertSession(generation);
  if (response.status === 401 && session) {
    if (session.access_token === usedToken) await refresh();
    response = await send();
  }
  assertSession(generation);
  if (!response.ok) throw await failure(response);
  const result = await (raw ? response.text() : response.json());
  assertSession(generation);
  return result as T;
}
export async function login(email: string, password: string) {
  const response = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  if (!response.ok) throw await failure(response);
  const value: Session = await response.json();
  setSession(value);
  return value;
}
export async function logout() {
  try {
    await request("/auth/logout", { method: "POST" });
  } finally {
    setSession(null);
  }
}
