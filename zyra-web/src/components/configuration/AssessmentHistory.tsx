import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { configurationApi as api } from "../../api/configuration.api";
import { assessmentsApi } from "../../api/assessments.api";
import { ticketsApi } from "../../api/tickets.api";
import type { Page, TicketSummary } from "../../types/api";
import type { AssessmentSummary } from "../../types/configuration";
import { assessmentLabel } from "../../types/ticket-table";
import { checkLabels } from "../../types/ticket-search";
import { receivedTimestamp } from "../../types/configuration-validation";
import { Button, Notice } from "../ui/Controls";
import { ListPagination } from "./ConfigurationControls";
import styles from "./Configuration.module.css";
export function AssessmentHistory({ databaseId }: { databaseId: string }) {
  const [data, setData] = useState<Page<AssessmentSummary> | null>(null);
  const [page, setPage] = useState(1);
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<string | null>(null);
  useEffect(() => {
    const abort = new AbortController();
    setData(null);
    setError("");
    api
      .assessments(databaseId, page, result, abort.signal)
      .then(setData)
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [databaseId, page, result]);
  return (
    <>
      <div className={styles.actions}>
        <label>
          Result{" "}
          <select
            value={result}
            onChange={(e) => {
              setResult(e.target.value);
              setPage(1);
            }}
          >
            <option value="">All results</option>
            <option value="passed">Passed</option>
            <option value="failed">Failed</option>
          </select>
        </label>
      </div>
      {error && <Notice error>{error}</Notice>}
      {!data ? (
        <Notice>Loading assessments…</Notice>
      ) : (
        <>
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Received / report time</th>
                  <th>Type</th>
                  <th>Result</th>
                  <th>Issues</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {data.items.map((item) => (
                  <tr key={item.id}>
                    <td>
                      {receivedTimestamp(item.received_at)}
                      {item.report_time && (
                        <div className={styles.muted}>
                          ReportOn: {item.report_time}
                        </div>
                      )}
                    </td>
                    <td>{assessmentLabel(item.type)}</td>
                    <td className={styles.result}>
                      {item.result.replaceAll("_", " ")}
                    </td>
                    <td>
                      <div className={styles.ticketLinks}>
                        {item.tickets.map((ticket) => (
                          <Link
                            key={ticket.id}
                            to="/tickets/$ticketId"
                            params={{ ticketId: ticket.id }}
                          >
                            #{ticket.number} · {ticket.title}
                          </Link>
                        ))}
                        {!item.ticket_count && <span>No issues created</span>}
                        {item.ticket_count > item.tickets.length && (
                          <Button onClick={() => setSelected(item.id)}>
                            View all {item.ticket_count} issues
                          </Button>
                        )}
                      </div>
                    </td>
                    <td>
                      <Button onClick={() => setSelected(item.id)}>
                        View assessment
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!data.items.length && <p>No assessments match this filter.</p>}
          <ListPagination
            page={page}
            total={data.total}
            count={data.items.length}
            onChange={setPage}
          />
        </>
      )}
      {selected && (
        <AssessmentDetail
          key={selected}
          id={selected}
          onClose={() => setSelected(null)}
        />
      )}
    </>
  );
}
function AssessmentDetail({
  id,
  onClose,
}: {
  id: string;
  onClose: () => void;
}) {
  const [metadata, setMetadata] = useState<AssessmentDetails | null>(null);
  const [tickets, setTickets] = useState<Page<TicketSummary> | null>(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  useEffect(() => {
    let live = true;
    setMetadata(null);
    assessmentsApi
      .detail(id)
      .then((v) => {
        if (live) setMetadata(v as unknown as AssessmentDetails);
      })
      .catch((e) => {
        if (live) setError(e.message);
      });
    return () => {
      live = false;
    };
  }, [id]);
  useEffect(() => {
    const abort = new AbortController();
    setTickets(null);
    ticketsApi
      .list(
        new URLSearchParams({
          assessment_id: id,
          limit: "50",
          page: String(page),
        }),
        abort.signal,
      )
      .then(setTickets)
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [id, page]);
  return (
    <section className={styles.panel}>
      <div className={styles.row}>
        <h2>Assessment details</h2>
        <Button onClick={onClose}>Close details</Button>
      </div>
      {error && <Notice error>{error}</Notice>}
      {metadata ? (
        <>
          <dl className={styles.identity}>
            <div>
              <dt>Result</dt>
              <dd>{String(metadata.result || "unknown")}</dd>
            </div>
            <div>
              <dt>Unique ID</dt>
              <dd>{String(metadata.unique_id || "Not present in report")}</dd>
            </div>
            <div>
              <dt>Subject</dt>
              <dd>{String(metadata.subject || "Not recorded")}</dd>
            </div>
          </dl>
          <h3>Check results</h3>
          <ul className={styles.list}>
            {metadata.results?.map((finding, index) => (
              <li key={index}>
                <strong>
                  {checkLabels[finding.check_type] || finding.check_type}
                </strong>{" "}
                · {finding.status.replaceAll("_", " ")}
                <p>{finding.summary}</p>
                {finding.resource && <p>Resource: {finding.resource}</p>}
                {finding.value !== undefined && (
                  <p>
                    Observed: {finding.value}
                    {finding.threshold !== undefined
                      ? ` · Threshold: ${finding.threshold}`
                      : ""}
                  </p>
                )}
                {finding.evidence && (
                  <details>
                    <summary>Evidence</summary>
                    <pre className={styles.evidence}>{finding.evidence}</pre>
                  </details>
                )}
              </li>
            ))}
          </ul>
        </>
      ) : (
        <Notice>Loading assessment details…</Notice>
      )}
      <h3>Generated issues</h3>
      {tickets ? (
        <>
          <ul className={styles.list}>
            {tickets.items.map((ticket) => (
              <li key={ticket.id}>
                <Link to="/tickets/$ticketId" params={{ ticketId: ticket.id }}>
                  #{ticket.number} · {ticket.title}
                </Link>
              </li>
            ))}
          </ul>
          {!tickets.total && <p>No issues created.</p>}
          <ListPagination
            page={page}
            total={tickets.total}
            count={tickets.items.length}
            onChange={setPage}
          />
        </>
      ) : (
        <Notice>Loading issue links…</Notice>
      )}
    </section>
  );
}
interface AssessmentDetails {
  result: string;
  unique_id: string;
  subject: string;
  results: {
    check_type: string;
    status: string;
    summary: string;
    resource?: string;
    evidence: string;
    value?: number;
    threshold?: number;
  }[];
}
