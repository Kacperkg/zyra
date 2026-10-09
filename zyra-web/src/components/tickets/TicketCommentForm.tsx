import { Button } from "../ui/Controls";
import { ActionIcon } from "../ui/ActionIcon";
import { RichEditor } from "../editor/RichEditor";
import { hasContent, type RichContent } from "../editor/content";
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
  comment: RichContent;
  busy: boolean;
  onComment: (value: RichContent) => void;
  onAction: (action: TicketAction) => void;
}) {
  return (
    <form
      className={styles.composer}
      onSubmit={(event) => {
        event.preventDefault();
        if (!busy && hasContent(comment)) onAction("comments");
      }}
    >
      <h2>Add a comment</h2>
      <RichEditor
        disabled={busy}
        onChange={onComment}
        initialContent={comment}
      />
      <div className={styles.actions}>
        <Button type="submit" primary disabled={busy || !hasContent(comment)}>
          Comment
        </Button>
        {status === "open" ? (
          <>
            <Button
              type="button"
              disabled={busy || !hasContent(comment)}
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
            <ActionIcon name="reopen" /> Reopen
          </Button>
        )}
      </div>
      {status === "closed" && (
        <small>Adding a comment keeps this ticket closed.</small>
      )}
    </form>
  );
}
