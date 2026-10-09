import { Link } from "@tanstack/react-router";
import type { TicketDetail } from "../../types/api";
import { StatusBadge } from "../ui/StatusBadge";
import { Avatar } from "../ui/Avatar";
import styles from "./TicketContextPanel.module.css";
export function TicketContextPanel({ detail }: { detail: TicketDetail }) {
  return (
    <aside className={styles.panel}>
      <section>
        <h3>Client</h3>
        <p>{detail.client.name}</p>
        <h3>Database</h3>
        <p className={styles.mono}>{detail.database.name}</p>
        <small>Client and database pages are coming later.</small>
      </section>
      <section>
        <h3>Database notes</h3>
        <p>{detail.database.notes || "No database notes."}</p>
        <h3>Ticket notes</h3>
        <small>Ticket-specific notes will be added later.</small>
      </section>
      <section>
        <h3>Participants</h3>
        {detail.participants.length ? (
          detail.participants.map((person) => (
            <p key={person.id} className={styles.participant}>
              <Avatar user={person} />
              {person.name}
            </p>
          ))
        ) : (
          <small>No comments from team members yet.</small>
        )}
      </section>
      <section>
        <h3>Similar issues</h3>
        <small>
          Same client, database and check type. Newest open match first.
        </small>
        <div className={styles.similar}>
          {detail.similar.map((ticket) => (
            <Link
              key={ticket.id}
              to="/tickets/$ticketId"
              params={{ ticketId: ticket.id }}
            >
              <div>
                <strong>#{ticket.number}</strong>
                <StatusBadge status={ticket.status} />
              </div>
              <span>{ticket.title}</span>
              <small>{new Date(ticket.created_at).toLocaleString()}</small>
            </Link>
          ))}
        </div>
        {!detail.similar.length && <p>No matching issues.</p>}
      </section>
    </aside>
  );
}
