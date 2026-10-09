import { request } from "./client";
import type { Page, SavedTicket, SavedTicketSummary } from "../types/api";
export const listSavedTickets = (
  q: string,
  page: number,
  signal?: AbortSignal,
) =>
  request<Page<SavedTicketSummary>>(
    `/users/me/saved-tickets?${new URLSearchParams({ q, page: String(page), limit: "50" })}`,
    { signal },
  );
export const saveTicket = (id: string, title?: string) =>
  request<SavedTicket>(`/users/me/saved-tickets/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: JSON.stringify(title === undefined ? {} : { title }),
  });
export const unsaveTicket = (id: string) =>
  request<{ ok: boolean }>(
    `/users/me/saved-tickets/${encodeURIComponent(id)}`,
    { method: "DELETE" },
  );
