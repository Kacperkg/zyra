import { useEffect, useRef, useState } from "react";
import { Link, useParams } from "@tanstack/react-router";
import { ticketsApi } from "../../api/tickets.api";
import {
  assessmentsApi,
  type AssessmentMetadata,
} from "../../api/assessments.api";
import { useAuth } from "../../contexts/AuthContext";
import type { TicketDetail, TicketEvent } from "../../types/api";
import { checkLabels } from "../../types/ticket-search";
import { Button, Notice, Status } from "../../components/ui/Controls";
import { Tabs } from "../../components/ui/Tabs";
import { TicketContextPanel } from "../../components/tickets/TicketContextPanel";
import { TicketTimeline } from "../../components/tickets/TicketTimeline";
import {
  TicketCommentForm,
  type TicketAction,
} from "../../components/tickets/TicketCommentForm";
import styles from "./TicketPage.module.css";

export function TicketPage() {
  const { ticketId } = useParams({ strict: false });
  // Remount on navigation so pending actions cannot change another ticket's draft.
  return <TicketScreen key={ticketId} id={ticketId || ""} />;
}

function TicketScreen({ id }: { id: string }) {
  const { session } = useAuth();
  const [detail, setDetail] = useState<TicketDetail | null>(null);
  const [events, setEvents] = useState<TicketEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [eventPage, setEventPage] = useState(1);
  const [loaded, setLoaded] = useState(0);
  const [secondLoaded, setSecondLoaded] = useState(false);
  const [tab, setTab] = useState<"discussion" | "raw">("discussion");
  const [raw, setRaw] = useState<{
    body: string;
    metadata: AssessmentMetadata;
  } | null>(null);
  const [rawError, setRawError] = useState("");
  const [rawRetry, setRawRetry] = useState(0);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [comment, setComment] = useState("");
  const [version, setVersion] = useState(0);
  const sentinel = useRef<HTMLDivElement>(null);
  const generation = useRef(0);
  const loadingEvents = useRef<number | null>(null);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  useEffect(() => {
    const current = ++generation.current;
    loadingEvents.current = null;
    setLoading(true);
    setError("");
    setSecondLoaded(false);
    Promise.all([
      ticketsApi.detail(id),
      ticketsApi.events(id, (eventPage - 1) * 2 + 1),
    ])
      .then(([data, timeline]) => {
        if (current !== generation.current) return;
        setDetail(data);
        setEvents(timeline.items);
        setTotal(timeline.total);
        setLoaded(timeline.items.length);
      })
      .catch((e) => {
        if (current === generation.current) setError(e.message);
      })
      .finally(() => {
        if (current === generation.current) setLoading(false);
      });
    return () => {
      generation.current++;
    };
  }, [id, version, eventPage]);

  useEffect(() => {
    let alive = true;
    if (tab === "raw" && !raw && detail?.ticket.assessment_id) {
      setRawError("");
      Promise.all([
        ticketsApi.raw(detail.ticket.assessment_id),
        assessmentsApi.detail(detail.ticket.assessment_id),
      ])
        .then(([body, metadata]) => {
          if (alive) setRaw({ body, metadata });
        })
        .catch((e) => {
          if (alive) setRawError(e.message);
        });
    }
    return () => {
      alive = false;
    };
  }, [tab, raw, detail?.ticket.assessment_id, rawRetry]);

  async function loadMore() {
    if (
      loadingEvents.current !== null ||
      secondLoaded ||
      loaded !== 50 ||
      (eventPage - 1) * 100 + loaded >= total
    )
      return;
    const current = generation.current;
    loadingEvents.current = current;
    try {
      const data = await ticketsApi.events(id, (eventPage - 1) * 2 + 2);
      if (current !== generation.current) return;
      setEvents((old) => [
        ...old,
        ...data.items.filter((event) => !old.some((x) => x.id === event.id)),
      ]);
      setLoaded(50 + data.items.length);
      setSecondLoaded(true);
      setTotal(data.total);
    } catch (e) {
      if (current === generation.current) setError((e as Error).message);
    } finally {
      if (loadingEvents.current === current) loadingEvents.current = null;
    }
  }
  useEffect(() => {
    if (!sentinel.current || loading || error) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) void loadMore();
      },
      { rootMargin: "100px" },
    );
    observer.observe(sentinel.current);
    return () => observer.disconnect();
  }, [loaded, secondLoaded, total, loading, tab, error, eventPage]);

  async function action(name: TicketAction) {
    if (
      busy ||
      ((name === "comments" || name === "comment-and-close") && !comment.trim())
    )
      return;
    setBusy(true);
    setError("");
    try {
      await ticketsApi.action(
        id,
        name,
        name === "comments" || name === "comment-and-close"
          ? comment
          : undefined,
      );
      if (!mounted.current) return;
      setComment("");
      setEventPage(1);
      setVersion((x) => x + 1);
    } catch (e) {
      if (mounted.current) setError((e as Error).message);
    } finally {
      if (mounted.current) setBusy(false);
    }
  }

  if (loading && !detail) return <Notice>Loading ticket…</Notice>;
  if (!detail)
    return (
      <Notice error>
        {error || "Ticket unavailable."}
        <Button onClick={() => setVersion((x) => x + 1)}>Retry</Button>
      </Notice>
    );
  const ticket = detail.ticket;
  return (
    <>
      <Link to="/issues/$status" params={{ status: ticket.status }}>
        ← {ticket.status === "closed" ? "Closed" : "Open"} Oracle issues
      </Link>
      <div className={styles.heading}>
        <div>
          <p className={styles.eyebrow}>
            TICKET #{ticket.number} ·{" "}
            {checkLabels[ticket.check_type] || ticket.check_type}
          </p>
          <h1>{ticket.title}</h1>
          <p>
            {detail.client.name} / {detail.database.name}
          </p>
          <p>
            Created{" "}
            <time dateTime={ticket.created_at}>
              {new Date(ticket.created_at).toLocaleString()}
            </time>
          </p>
          <p className={styles.server}>
            Server: {ticket.hostname || "Not recorded"}
            {ticket.ip ? ` · ${ticket.ip}` : ""}
          </p>
        </div>
        <Status value={ticket.status} />
      </div>
      <Tabs
        id="ticket-content"
        label="Ticket content"
        value={tab}
        onChange={setTab}
        items={[
          { value: "discussion", label: "Discussion" },
          { value: "raw", label: "Raw email" },
        ]}
      />
      {error && (
        <Notice error>
          {error}{" "}
          <Button onClick={() => setVersion((x) => x + 1)}>Retry</Button>
        </Notice>
      )}
      <div className={styles.layout}>
        <section
          id="ticket-content-panel"
          role="tabpanel"
          aria-labelledby={`ticket-content-${tab}`}
          tabIndex={0}
        >
          {tab === "raw" ? (
            ticket.assessment_id ? (
              rawError ? (
                <Notice error>
                  {rawError}{" "}
                  <Button onClick={() => setRawRetry((x) => x + 1)}>
                    Retry raw email
                  </Button>
                </Notice>
              ) : raw ? (
                <>
                  <dl className={styles.metadata}>
                    <dt>Subject</dt>
                    <dd>{raw.metadata.subject || "Not recorded"}</dd>
                    <dt>Sender</dt>
                    <dd>{raw.metadata.sender || "Not recorded"}</dd>
                    <dt>Received</dt>
                    <dd>
                      {new Date(raw.metadata.received_at).toLocaleString()}
                    </dd>
                  </dl>
                  <pre className={styles.raw}>{raw.body}</pre>
                </>
              ) : (
                <Notice>Loading raw email…</Notice>
              )
            ) : (
              <Notice>No email was received for this ticket.</Notice>
            )
          ) : (
            <>
              {loading ? (
                <Notice>Loading events…</Notice>
              ) : (
                <TicketTimeline
                  events={events}
                  participants={detail.participants}
                  currentUser={session?.user}
                />
              )}
              <div ref={sentinel} className={styles.sentinel}>
                {!secondLoaded &&
                loaded === 50 &&
                (eventPage - 1) * 100 + loaded < total
                  ? "Scroll for more events"
                  : ""}
              </div>
              {total > 100 && (
                <div className={styles.eventPages}>
                  <Button
                    disabled={eventPage === 1 || loading || busy}
                    onClick={() => {
                      setEventPage((x) => x - 1);
                      window.scrollTo(0, 0);
                    }}
                  >
                    Previous events
                  </Button>
                  <span>
                    Page {eventPage} · {events.length} events
                  </span>
                  <Button
                    disabled={
                      loaded < 100 ||
                      eventPage * 100 >= total ||
                      loading ||
                      busy
                    }
                    onClick={() => {
                      setEventPage((x) => x + 1);
                      window.scrollTo(0, 0);
                    }}
                  >
                    Next events
                  </Button>
                </div>
              )}
              <TicketCommentForm
                status={ticket.status}
                comment={comment}
                busy={busy || loading}
                onComment={setComment}
                onAction={(name) => void action(name)}
              />
            </>
          )}
        </section>
        <TicketContextPanel detail={detail} />
      </div>
    </>
  );
}
