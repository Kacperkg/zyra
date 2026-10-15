import { request } from "./client";
import type {
  Page,
  TicketSummary,
  TicketDetail,
  TicketEvent,
  CommentInput,
} from "../types/api";
export const ticketsApi = {
  list: (query: URLSearchParams, signal?: AbortSignal) =>
    request<Page<TicketSummary>>(`/tickets?${query}`, { signal }),
  detail: (id: string, signal?: AbortSignal) =>
    request<TicketDetail>(`/tickets/${encodeURIComponent(id)}`, { signal }),
  events: (id: string, page: number, signal?: AbortSignal) =>
    request<Page<TicketEvent>>(
      `/tickets/${encodeURIComponent(id)}/events?page=${page}`,
      { signal },
    ),
  raw: (assessmentId: string) =>
    request<string>(
      `/assessments/${encodeURIComponent(assessmentId)}/raw-email`,
      {},
      true,
    ),
  action: (id: string, action: string, comment?: CommentInput) =>
    request(`/tickets/${encodeURIComponent(id)}/${action}`, {
      method: "POST",
      body: JSON.stringify(comment || {}),
    }),
  editComment: (id: string, commentId: string, input: CommentInput) =>
    request(
      `/tickets/${encodeURIComponent(id)}/comments/${encodeURIComponent(commentId)}`,
      { method: "PATCH", body: JSON.stringify(input) },
    ),
  deleteComment: (id: string, commentId: string, revision: number) =>
    request(
      `/tickets/${encodeURIComponent(id)}/comments/${encodeURIComponent(commentId)}?revision=${revision}`,
      { method: "DELETE" },
    ),
};
