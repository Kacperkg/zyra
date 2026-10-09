import type { TicketSearch } from "./ticket-search";
// Each UI page requests two fixed batches. Detail payloads are never fetched here.
export function buildTicketQuery(
  status: string | undefined,
  search: TicketSearch,
  batch: 1 | 2,
) {
  const page = search.page || 1;
  const params = new URLSearchParams({
    status: status || "open",
    limit: "50",
    page: String((page - 1) * 2 + batch),
    sort: search.sort || "created_at",
    order: search.order === "asc" ? "asc" : "desc",
  });
  if (search.q) params.set("q", search.q);
  if (search.check) params.set("check_type", search.check);
  for (const key of [
    "client_id",
    "database_id",
    "assessment_type",
    "number",
    "created_from",
    "created_to",
  ] as const) {
    if (search[key]) params.set(key, search[key]!);
  }
  return params;
}
