import { request } from "./client";
import type { NotificationSummary, Page } from "../types/api";
export const notificationsApi = {
  list: (page = 1, signal?: AbortSignal) =>
    request<Page<NotificationSummary>>(
      `/users/me/notifications?page=${page}&limit=20`,
      { signal },
    ),
  unread: (signal?: AbortSignal) =>
    request<{ count: number }>("/users/me/notifications/unread-count", {
      signal,
    }),
  read: (id: string) =>
    request<{ ok: boolean }>(
      `/users/me/notifications/${encodeURIComponent(id)}/read`,
      { method: "PATCH" },
    ),
  readAll: () =>
    request<{ ok: boolean }>("/users/me/notifications/read-all", {
      method: "POST",
    }),
};
