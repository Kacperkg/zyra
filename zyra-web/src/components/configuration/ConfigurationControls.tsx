import { Button } from "../ui/Controls";
import styles from "./Configuration.module.css";
export function ListPagination({
  page,
  total,
  count,
  onChange,
}: {
  page: number;
  total: number;
  count: number;
  onChange: (page: number) => void;
}) {
  return (
    <div className={styles.actions}>
      <span className={styles.muted}>
        {total
          ? `${(page - 1) * 50 + 1}–${(page - 1) * 50 + count} of ${total}`
          : "0 results"}
      </span>
      <Button disabled={page === 1} onClick={() => onChange(page - 1)}>
        Previous
      </Button>
      <Button disabled={page * 50 >= total} onClick={() => onChange(page + 1)}>
        Next
      </Button>
    </div>
  );
}
export function ConfigurationSave({
  busy,
  dirty,
  onCancel,
}: {
  busy: boolean;
  dirty: boolean;
  onCancel: () => void;
}) {
  return (
    <div className={styles.actions}>
      <Button type="submit" primary disabled={busy || !dirty}>
        {busy ? "Saving…" : "Save changes"}
      </Button>
      <Button type="button" disabled={busy || !dirty} onClick={onCancel}>
        Cancel changes
      </Button>
      {dirty && <small className={styles.muted}>Unsaved changes</small>}
    </div>
  );
}
