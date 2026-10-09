import { request } from "./client";
import type { User } from "../types/api";
export type ProfileInput = Partial<
  Pick<User, "name" | "email" | "avatar_url" | "theme" | "appearance">
>;
export const usersApi = {
  me: (signal?: AbortSignal) => request<User>("/users/me", { signal }),
  update: (input: ProfileInput) =>
    request<User>("/users/me", {
      method: "PATCH",
      body: JSON.stringify(input),
    }),
  mentionOptions: (q: string, signal?: AbortSignal) =>
    request<{ items: Pick<User, "id" | "name" | "avatar_url">[] }>(
      `/users/mention-options?${new URLSearchParams({ q, limit: "20" })}`,
      { signal },
    ),
};
