import { request } from "./client";
import type {
  Page,
  TicketSummary,
  TicketDetail,
  TicketEvent,
} from "../types/api";
export const ticketsApi = {
  list: (query: URLSearchParams, signal?: AbortSignal) =>
    request<Page<TicketSummary>>(`/tickets?${query}`, { signal }),
  detail: (id: string) =>
    request<TicketDetail>(`/tickets/${encodeURIComponent(id)}`),
  events: (id: string, page: number) =>
    request<Page<TicketEvent>>(
      `/tickets/${encodeURIComponent(id)}/events?page=${page}`,
    ),
  raw: (assessmentId: string) =>
    request<string>(
      `/assessments/${encodeURIComponent(assessmentId)}/raw-email`,
      {},
      true,
    ),
  action: (id: string, action: string, comment?: string) =>
    request(`/tickets/${encodeURIComponent(id)}/${action}`, {
      method: "POST",
      body: JSON.stringify({ comment }),
    }),
};
