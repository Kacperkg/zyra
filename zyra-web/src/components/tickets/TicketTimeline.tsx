import type { TicketEvent, User } from "../../types/api";
import styles from "./TicketTimeline.module.css";
export function TicketTimeline({
  events,
  participants,
  currentUser,
}: {
  events: TicketEvent[];
  participants: User[];
  currentUser?: User;
}) {
  const names = new Map(participants.map((user) => [user.id, user.name]));
  if (currentUser) names.set(currentUser.id, currentUser.name);
  return (
    <div className={styles.timeline}>
      {events.map((event) => (
        <article key={event.id} className={styles.event}>
          <header>
            <strong>
              {event.type === "system_findings"
                ? "Zyra system"
                : names.get(event.user_id) || "Team member"}
            </strong>
            <span>
              {event.type.replaceAll("_", " ")} ·{" "}
              <time dateTime={event.created_at}>
                {new Date(event.created_at).toLocaleString()}
              </time>
            </span>
          </header>
          {event.comment && <p>{event.comment}</p>}
        </article>
      ))}
    </div>
  );
}
