import { Link } from "@tanstack/react-router";
import type { TicketSummary } from "../../types/api";
import { checkLabels } from "../../types/ticket-search";
import {
  assessmentLabel,
  ticketColumns,
  ticketTimestamp,
} from "../../types/ticket-table";
import { StatusBadge } from "../ui/StatusBadge";
import styles from "./TicketTable.module.css";
export function TicketTable({
  tickets,
  appearance = "modern",
  sort,
  order,
  onSort,
}: {
  tickets: TicketSummary[];
  appearance?: "modern" | "classic";
  sort?: string;
  order?: string;
  onSort?: (sort: string) => void;
}) {
  const headers = ticketColumns(appearance);
  const classic = appearance === "classic";
  return (
    <div
      className={`${styles.wrapper} ${classic ? styles.classic : ""}`}
      data-appearance={appearance}
      role="region"
      aria-label="Oracle ticket table"
      tabIndex={0}
    >
      <table>
        <caption className={styles.caption}>
          Oracle tickets — {appearance} view
        </caption>
        <thead>
          <tr>
            {headers.map(({ id, label, sort: sortKey }) => (
              <th
                key={id}
                scope="col"
                data-column={id}
                aria-sort={
                  sortKey && sort === sortKey
                    ? order === "asc"
                      ? "ascending"
                      : "descending"
                    : undefined
                }
              >
                {onSort && sortKey ? (
                  <button
                    type="button"
                    className={styles.sort}
                    onClick={() => onSort(sortKey)}
                    aria-label={`Sort by ${label}`}
                  >
                    {label}
                    <span aria-hidden="true">
                      {sort === sortKey ? (order === "asc" ? "↑" : "↓") : "↕"}
                    </span>
                  </button>
                ) : (
                  label
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => {
            const timestamp = ticketTimestamp(ticket.created_at, appearance);
            return (
              <tr key={ticket.id}>
                <td>
                  <Link
                    to="/tickets/$ticketId"
                    params={{ ticketId: ticket.id }}
                    aria-label={`Open ticket ${ticket.number}: ${ticket.title}`}
                  >
                    {classic ? ticket.number : `#${ticket.number}`}
                  </Link>
                </td>
                <td>{checkLabels[ticket.check_type] || ticket.check_type}</td>
                <td>
                  <time dateTime={ticket.created_at}>{timestamp.date}</time>
                </td>
                <td>
                  <time dateTime={ticket.created_at}>{timestamp.time}</time>
                </td>
                <td>{ticket.client_name}</td>
                <td className={classic ? styles.database : styles.mono}>
                  {ticket.database_name}
                </td>
                <td>{assessmentLabel(ticket.assessment_type)}</td>
                {!classic && (
                  <td>
                    <StatusBadge status={ticket.status} />
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
