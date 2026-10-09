import type { TicketSummary } from "./api";
export interface TicketColumn {
  id: string;
  label: string;
  sort?: string;
}
export function ticketColumns(
  appearance: "modern" | "classic",
): TicketColumn[] {
  const classic = appearance === "classic";
  return [
    { id: "number", label: classic ? "#" : "Ticket", sort: "number" },
    { id: "check_type", label: "Check type", sort: "check_type" },
    {
      id: "created_date",
      label: classic ? "Created Date" : "Created date",
      sort: "created_at",
    },
    { id: "created_time", label: classic ? "Created Time" : "Time" },
    { id: "client", label: "Client", sort: "client" },
    { id: "database", label: "Database", sort: "database" },
    {
      id: "assessment_type",
      label: classic ? "Assessment Type" : "Assessment",
      sort: "assessment_type",
    },
    ...(!classic ? [{ id: "status", label: "Status", sort: "status" }] : []),
  ];
}
export function ticketTimestamp(
  value: string,
  appearance: "modern" | "classic",
) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime()))
    return { date: "Unavailable", time: "Unavailable" };
  return {
    date:
      appearance === "classic"
        ? [
            date.getFullYear(),
            String(date.getMonth() + 1).padStart(2, "0"),
            String(date.getDate()).padStart(2, "0"),
          ].join("-")
        : date.toLocaleDateString(),
    time:
      appearance === "classic"
        ? [date.getHours(), date.getMinutes(), date.getSeconds()]
            .map((part) => String(part).padStart(2, "0"))
            .join(":")
        : date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
  };
}
export function assessmentLabel(value: TicketSummary["assessment_type"]) {
  const labels: Record<string, string> = {
    daily_check: "Daily check",
    missing: "Missing email check",
    missing_email: "Missing email check",
    standby: "Standby check",
  };
  return labels[value] || value.replaceAll("_", " ");
}
