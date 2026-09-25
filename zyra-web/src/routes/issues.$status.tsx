import { createFileRoute, notFound } from "@tanstack/react-router";
import { TicketListPage } from "../pages/tickets/TicketListPage";
import type { TicketSearch } from "../types/ticket-search";
export const Route = createFileRoute("/issues/$status")({
  beforeLoad: ({ params }) => {
    if (!["open", "closed"].includes(params.status)) throw notFound();
  },
  validateSearch: (search: Record<string, unknown>): TicketSearch => {
    const result: TicketSearch = {
      page: Number.isFinite(Number(search.page))
        ? Math.max(1, Math.floor(Number(search.page) || 1))
        : 1,
    };
    for (const key of [
      "q",
      "check",
      "sort",
      "order",
      "client_id",
      "database_id",
      "assessment_type",
      "number",
      "created_from",
      "created_to",
    ] as const) {
      if (typeof search[key] === "string") result[key] = search[key] as string;
    }
    return result;
  },
  component: TicketListPage,
});
