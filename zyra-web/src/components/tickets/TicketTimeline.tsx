import { useState } from "react";
import type { TicketEvent, User, CommentContent } from "../../types/api";
import { ticketsApi } from "../../api/tickets.api";
import { RichEditor } from "../editor/RichEditor";
import { RichContent } from "../editor/RichContent";
import { hasContent } from "../editor/content";
import { Button, Notice } from "../ui/Controls";
import { Avatar } from "../ui/Avatar";
import { ActionIcon } from "../ui/ActionIcon";
import styles from "./TicketTimeline.module.css";
function Findings({ text }: { text: string }) {
  const [heading, ...findings] = text.split(/\n- /);
  return (
    <div className={styles.findings}>
      <h3>{heading}</h3>
      {findings.map((finding, index) => (
        <section key={index}>
          {finding
            .split(
              /; (?=Resource:|Finding:|Reported |Configured |Maximum |Minimum |The received)/,
            )
            .map((line, i) => (
              <p key={i}>{line}</p>
            ))}
        </section>
      ))}
    </div>
  );
}
function CommentEvent({
  event,
  person,
  names,
  own,
  ticketId,
  onChanged,
}: {
  event: TicketEvent;
  person?: User;
  names: Record<string, string>;
  own: boolean;
  ticketId: string;
  onChanged: () => void;
}) {
  const initial = event.content || {
    blocks: [
      {
        type: "paragraph" as const,
        children: [{ type: "text" as const, text: event.comment || "" }],
      },
    ],
  };
  const [editing, setEditing] = useState(false);
  const [content, setContent] = useState<CommentContent>(initial);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);
  async function change(remove: boolean) {
    if (busy || !event.comment_id || !event.revision) return;
    setBusy(true);
    setError("");
    try {
      if (remove)
        await ticketsApi.deleteComment(
          ticketId,
          event.comment_id,
          event.revision,
        );
      else
        await ticketsApi.editComment(ticketId, event.comment_id, {
          schema_version: 1,
          content,
          revision: event.revision,
        });
      setEditing(false);
      setConfirmDelete(false);
      onChanged();
    } catch (e) {
      setError(
        (e as Error).message +
          " Refresh the discussion if someone changed this comment.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <article className={styles.event}>
      <header>
        <div className={styles.author}>
          {person && <Avatar user={person} />}
          <strong>
            {event.type === "system_findings"
              ? "Zyra system"
              : names[event.user_id] || "Team member"}
          </strong>
        </div>
        <div className={styles.headerActions}>
          <span>
            <time dateTime={event.created_at}>
              {new Date(event.created_at).toLocaleString()}
            </time>
            {event.edited_at && (
              <span title={new Date(event.edited_at).toLocaleString()}>
                {" "}
                · Edited
              </span>
            )}
          </span>
          {own && event.comment_id && !editing && !confirmDelete && (
            <div className={styles.iconActions}>
              <Button
                className={styles.iconButton}
                title="Edit comment"
                aria-label="Edit comment"
                disabled={busy}
                onClick={() => {
                  setContent(initial);
                  setEditing(true);
                }}
              >
                <ActionIcon name="edit" />
              </Button>
              <Button
                className={styles.iconButton}
                title="Delete comment"
                aria-label="Delete comment"
                disabled={busy}
                onClick={() => setConfirmDelete(true)}
              >
                <ActionIcon name="delete" />
              </Button>
            </div>
          )}
        </div>
      </header>
      {event.type === "system_findings" ? (
        <Findings text={event.comment || ""} />
      ) : editing ? (
        <RichEditor
          key={event.revision}
          initialContent={initial}
          onChange={setContent}
          names={names}
          disabled={busy}
          label="Edit comment"
        />
      ) : event.content ? (
        <RichContent content={event.content} names={names} />
      ) : event.comment ? (
        <RichContent content={initial} names={names} />
      ) : (
        <p>{event.type.replaceAll("_", " ")}</p>
      )}
      {error && <Notice error>{error}</Notice>}
      {own && event.comment_id && (editing || confirmDelete) && (
        <div className={styles.actions}>
          {editing ? (
            <>
              <Button
                disabled={busy || !hasContent(content)}
                onClick={() => void change(false)}
              >
                Save changes
              </Button>
              <Button
                disabled={busy}
                onClick={() => {
                  setEditing(false);
                  setContent(initial);
                  setError("");
                }}
              >
                Cancel
              </Button>
            </>
          ) : confirmDelete ? (
            <>
              <span>Delete this comment permanently?</span>
              <Button disabled={busy} onClick={() => void change(true)}>
                Delete comment
              </Button>
              <Button disabled={busy} onClick={() => setConfirmDelete(false)}>
                Cancel
              </Button>
            </>
          ) : null}
        </div>
      )}
    </article>
  );
}
export function TicketTimeline({
  events,
  participants,
  currentUser,
  ticketId,
  onChanged,
}: {
  events: TicketEvent[];
  participants: User[];
  currentUser?: User;
  ticketId: string;
  onChanged: () => void;
}) {
  const people = new Map(participants.map((user) => [user.id, user]));
  if (currentUser) people.set(currentUser.id, currentUser);
  const names = Object.fromEntries(
    [...people.values()].map((user) => [user.id, user.name]),
  );
  for (const event of events)
    for (const user of event.mention_users || []) names[user.id] = user.name;
  return (
    <div className={styles.timeline}>
      {events.map((event) =>
        event.type === "close" || event.type === "reopen" ? (
          <div className={styles.activity} key={event.id}>
            <ActionIcon name={event.type === "close" ? "close" : "reopen"} />
            <span>
              <strong>{names[event.user_id] || "Team member"}</strong>{" "}
              {event.type === "close"
                ? "closed this ticket."
                : "reopened this ticket."}
            </span>
            <time dateTime={event.created_at}>
              {new Date(event.created_at).toLocaleString()}
            </time>
          </div>
        ) : (
          <CommentEvent
            key={`${event.id}-${event.revision || 0}`}
            event={event}
            person={people.get(event.user_id)}
            names={names}
            own={!!currentUser && event.user_id === currentUser.id}
            ticketId={ticketId}
            onChanged={onChanged}
          />
        ),
      )}
    </div>
  );
}
