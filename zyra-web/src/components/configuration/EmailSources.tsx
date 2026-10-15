import { useEffect, useMemo, useState } from "react";
import { Link } from "@tanstack/react-router";
import type { Page } from "../../types/api";
import type { EmailSource } from "../../types/configuration";
import { configurationApi as api } from "../../api/configuration.api";
import { Button, Notice } from "../ui/Controls";
import { ListPagination } from "./ConfigurationControls";
import { assessmentLabel } from "../../types/ticket-table";
import styles from "./Configuration.module.css";
export function EmailSources({
  databaseId,
  editable,
  schedules = false,
}: {
  databaseId: string;
  editable: boolean;
  schedules?: boolean;
}) {
  const [data, setData] = useState<Page<EmailSource> | null>(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);
  // Both database tabs share this instance. Leaving for source editing unmounts
  // it, so returning after a save always starts with fresh configuration.
  const pages = useMemo(
    () => new Map<number, Page<EmailSource>>(),
    [databaseId],
  );
  useEffect(() => {
    const abort = new AbortController();
    setError("");
    const cached = pages.get(page);
    setData(cached || null);
    if (cached) return;
    api
      .sources(databaseId, page, abort.signal)
      .then((result) => {
        if (abort.signal.aborted) return;
        pages.set(page, result);
        setData(result);
      })
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [databaseId, page, pages, revision]);
  return (
    <>
      {schedules && (
        <Notice>
          Saved expected-email windows only. Live mailbox reading and background
          missing-email monitoring are not connected.
        </Notice>
      )}
      <Button
        onClick={() => {
          pages.clear();
          setRevision((value) => value + 1);
        }}
      >
        Refresh assessment emails
      </Button>
      {error && <Notice error>{error}</Notice>}
      {!data ? (
        !error && <Notice>Loading assessment email configurations…</Notice>
      ) : (
        <>
          <ul className={styles.list}>
            {data.items.map((source) => (
              <li key={source.id}>
                <Link
                  to="/databases/$databaseId/emails/$sourceId"
                  params={{ databaseId, sourceId: source.id }}
                >
                  {source.subject}
                </Link>
                <p className={styles.muted}>
                  {source.sender} → {source.mailbox || "Mailbox not configured"}{" "}
                  · {assessmentLabel(source.assessment_type)} ·{" "}
                  {source.enabled ? "Active" : "Inactive"}
                </p>
                {schedules && (
                  <>
                    <p>
                      {source.timezone} ·{" "}
                      {source.weekdays?.join(", ") || "No selected days"}
                    </p>
                    {source.schedules.map((window) => (
                      <p key={window.name}>
                        {window.name}: {window.start}–{window.end}
                      </p>
                    ))}
                    {!source.schedules.length && (
                      <p>No expected delivery windows configured.</p>
                    )}
                  </>
                )}
              </li>
            ))}
          </ul>
          {!data.total && <p>No assessment emails configured.</p>}
          <ListPagination
            page={page}
            total={data.total}
            count={data.items.length}
            onChange={setPage}
          />
        </>
      )}
      {editable && !schedules && (
        <Link
          to="/databases/$databaseId/emails/$sourceId"
          params={{ databaseId, sourceId: "new" }}
        >
          Add assessment email
        </Link>
      )}
    </>
  );
}
