import { request } from "./client";
import type { Page } from "../types/api";
import type {
  ClientRecord,
  DatabaseRecord,
  CheckSettings,
  EmailSource,
  AssessmentSummary,
} from "../types/configuration";
const path = (id: string) => encodeURIComponent(id);
export const configurationApi = {
  clients: (page = 1, q = "", signal?: AbortSignal) =>
    request<Page<ClientRecord>>(
      `/clients?limit=50&page=${page}&q=${encodeURIComponent(q)}`,
      { signal },
    ),
  client: (id: string, signal?: AbortSignal) =>
    request<ClientRecord>(`/clients/${path(id)}`, { signal }),
  saveClient: (id: string, input: Partial<ClientRecord>) =>
    request<ClientRecord>(id ? `/clients/${path(id)}` : "/clients", {
      method: id ? "PATCH" : "POST",
      body: JSON.stringify(input),
    }),
  archiveClient: (id: string) =>
    request(`/clients/${path(id)}`, { method: "DELETE" }),
  databases: (client: string, page = 1, signal?: AbortSignal) =>
    request<Page<DatabaseRecord>>(
      `/databases?client_id=${path(client)}&engine=oracle&limit=50&page=${page}`,
      { signal },
    ),
  database: (id: string, signal?: AbortSignal) =>
    request<DatabaseRecord>(`/databases/${path(id)}`, { signal }),
  saveDatabase: (id: string, input: Partial<DatabaseRecord>) =>
    request<DatabaseRecord>(id ? `/databases/${path(id)}` : "/databases", {
      method: id ? "PATCH" : "POST",
      body: JSON.stringify(input),
    }),
  archiveDatabase: (id: string) =>
    request(`/databases/${path(id)}`, { method: "DELETE" }),
  settings: (id: string, signal?: AbortSignal) =>
    request<CheckSettings>(`/databases/${path(id)}/check-settings`, { signal }),
  saveSettings: async (id: string, settings: CheckSettings) => {
    await request(`/databases/${path(id)}/check-settings`, {
      method: "PUT",
      body: JSON.stringify(settings),
    });
    return request<CheckSettings>(`/databases/${path(id)}/check-settings`);
  },
  sources: (id: string, page = 1, signal?: AbortSignal) =>
    request<Page<EmailSource>>(
      `/databases/${path(id)}/email-sources?limit=50&page=${page}`,
      { signal },
    ),
  source: (id: string, signal?: AbortSignal) =>
    request<EmailSource>(`/email-sources/${path(id)}`, { signal }),
  saveSource: (database: string, id: string, input: Partial<EmailSource>) =>
    request<EmailSource>(
      id
        ? `/email-sources/${path(id)}`
        : `/databases/${path(database)}/email-sources`,
      { method: id ? "PATCH" : "POST", body: JSON.stringify(input) },
    ),
  assessments: (id: string, page = 1, result = "", signal?: AbortSignal) =>
    request<Page<AssessmentSummary>>(
      `/databases/${path(id)}/assessments?limit=50&page=${page}${result ? `&result=${path(result)}` : ""}`,
      { signal },
    ),
};
