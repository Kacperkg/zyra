import type { Session } from "../types/api";
let session: Session | null = null;
let onChange: (value: Session | null) => void = () => {};
let refreshing: Promise<void> | null = null;
let revision = 0;
export function watchSession(listener: typeof onChange) {
  onChange = listener;
  return () => {
    onChange = () => {};
  };
}
export function setSession(value: Session | null) {
  revision++;
  refreshing = null;
  session = value;
  onChange(value);
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
  refreshing = (async () => {
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
    session = next;
    onChange(next);
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
