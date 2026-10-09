import { request } from "./client";
export interface MentionOption {
  id: string;
  name: string;
  avatar_url?: string;
}
export const mentionOptions = (q: string, signal?: AbortSignal) =>
  request<{ items: MentionOption[] }>(
    `/users/mention-options?q=${encodeURIComponent(q)}&limit=10`,
    { signal },
  ).then((data) => data.items);
