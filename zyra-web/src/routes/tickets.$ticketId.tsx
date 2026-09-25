import { createFileRoute } from "@tanstack/react-router";
import { TicketPage } from "../pages/tickets/TicketPage";
export const Route = createFileRoute("/tickets/$ticketId")({
  component: TicketPage,
});
