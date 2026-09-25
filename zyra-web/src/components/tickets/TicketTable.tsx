import { Link } from "@tanstack/react-router";
import type { TicketSummary } from "../../types/api";
import { checkLabels } from "../../types/ticket-search";
import { Status } from "../ui/Controls";
import styles from "./TicketTable.module.css";
export function TicketTable({ tickets }: { tickets: TicketSummary[] }) {
  return (
    <div className={styles.wrapper}>
      <table>
        <thead>
          <tr>
            <th>Ticket</th>
            <th>Check type</th>
            <th>Created date</th>
            <th>Time</th>
            <th>Client</th>
            <th>Database</th>
            <th>Assessment</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {tickets.map((ticket) => (
            <tr key={ticket.id}>
              <td>
                <Link to="/tickets/$ticketId" params={{ ticketId: ticket.id }}>
                  #{ticket.number}
                </Link>
              </td>
              <td>{checkLabels[ticket.check_type] || ticket.check_type}</td>
              <td>{new Date(ticket.created_at).toLocaleDateString()}</td>
              <td>
                {new Date(ticket.created_at).toLocaleTimeString([], {
                  hour: "2-digit",
                  minute: "2-digit",
                })}
              </td>
              <td>{ticket.client_name}</td>
              <td className={styles.mono}>{ticket.database_name}</td>
              <td>{ticket.assessment_type.replaceAll("_", " ")}</td>
              <td>
                <Status value={ticket.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
