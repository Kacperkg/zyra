import { Button, fieldClass } from "../ui/Controls";
import styles from "./TicketCommentForm.module.css";
export type TicketAction =
  "comments" | "comment-and-close" | "close" | "reopen";
export function TicketCommentForm({
  status,
  comment,
  busy,
  onComment,
  onAction,
}: {
  status: "open" | "closed";
  comment: string;
  busy: boolean;
  onComment: (value: string) => void;
  onAction: (action: TicketAction) => void;
}) {
  return (
    <form
      className={styles.composer}
      onSubmit={(event) => {
        event.preventDefault();
        if (!busy && comment.trim()) onAction("comments");
      }}
    >
      <h2>Add a comment</h2>
      <textarea
        className={fieldClass}
        aria-label="Comment"
        placeholder="What did you investigate or change?"
        rows={5}
        value={comment}
        disabled={busy}
        onChange={(event) => onComment(event.target.value)}
      />
      <small>
        Plain text for now. Rich text and attachments are coming later.
      </small>
      <div className={styles.actions}>
        <Button type="submit" primary disabled={busy || !comment.trim()}>
          Comment
        </Button>
        {status === "open" ? (
          <>
            <Button
              type="button"
              disabled={busy || !comment.trim()}
              onClick={() => onAction("comment-and-close")}
            >
              Comment and close
            </Button>
            <Button
              type="button"
              disabled={busy}
              onClick={() => onAction("close")}
            >
              Close without comment
            </Button>
          </>
        ) : (
          <Button
            type="button"
            disabled={busy}
            onClick={() => onAction("reopen")}
          >
            Reopen
          </Button>
        )}
      </div>
      {status === "closed" && (
        <small>Adding a comment keeps this ticket closed.</small>
      )}
    </form>
  );
}
