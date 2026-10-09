import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams, useSearch } from "@tanstack/react-router";
import { ticketsApi } from "../../api/tickets.api";
import { RemoteSelect } from "../../components/ui/RemoteSelect";
import { checkLabels, type TicketSearch } from "../../types/ticket-search";
import { buildTicketQuery } from "../../types/ticket-query";
import type { TicketSummary } from "../../types/api";
import { Button, Notice, fieldClass } from "../../components/ui/Controls";
import styles from "./TicketListPage.module.css";
import { TicketTable } from "../../components/tickets/TicketTable";
import { useAppearance } from "../../contexts/ThemeContext";
export function TicketListPage() {
  const { appearance } = useAppearance();
  const classic = appearance === "classic";
  const { status } = useParams({ strict: false });
  const search = useSearch({ strict: false }) as TicketSearch;
  const [filtersOpen, setFiltersOpen] = useState(false);
  const navigate = useNavigate();
  const [rows, setRows] = useState<TicketSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [loaded, setLoaded] = useState(0);
  const [secondLoaded, setSecondLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [version, setVersion] = useState(0);
  const [term, setTerm] = useState(search.q || "");
  const sentinel = useRef<HTMLDivElement>(null);
  const page = search.page || 1;
  const identity = `${status}:${JSON.stringify(search)}:${version}`;
  const generation = useRef(0);
  const fetching = useRef(false);
  function change(next: Partial<TicketSearch>) {
    void navigate({
      to: "/issues/$status",
      params: { status: status || "open" },
      search: { ...search, ...next },
    });
  }
  useEffect(() => setTerm(search.q || ""), [search.q]);
  useEffect(() => {
    const controller = new AbortController();
    const current = ++generation.current;
    fetching.current = true;
    setBusy(true);
    setError("");
    setRows([]);
    setLoaded(0);
    setSecondLoaded(false);
    const params = buildTicketQuery(status, search, 1);
    ticketsApi
      .list(params, controller.signal)
      .then((data) => {
        if (current !== generation.current) return;
        setRows(data.items);
        setTotal(data.total);
        setLoaded(data.items.length);
      })
      .catch((e) => {
        if (!controller.signal.aborted) setError(e.message);
      })
      .finally(() => {
        if (current === generation.current) {
          fetching.current = false;
          setBusy(false);
        }
      });
    return () => controller.abort();
  }, [identity]);
  async function more() {
    if (
      fetching.current ||
      secondLoaded ||
      loaded !== 50 ||
      (page - 1) * 100 + loaded >= total
    )
      return;
    fetching.current = true;
    setBusy(true);
    setError("");
    const current = generation.current;
    try {
      const params = buildTicketQuery(status, search, 2);
      const data = await ticketsApi.list(params);
      if (current !== generation.current) return;
      setRows((old) => [
        ...old,
        ...data.items.filter((row) => !old.some((x) => x.id === row.id)),
      ]);
      setLoaded(50 + data.items.length);
      setSecondLoaded(true);
      setTotal(data.total);
    } catch (e) {
      if (current === generation.current) setError((e as Error).message);
    } finally {
      if (current === generation.current) {
        fetching.current = false;
        setBusy(false);
      }
    }
  }
  useEffect(() => {
    const node = sentinel.current;
    if (!node || busy || error) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) void more();
      },
      { rootMargin: "120px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [loaded, busy, total, identity, error, secondLoaded]);
  const searchControls = (
    <>
      <label className={styles.searchField}>
        {classic && <span>Search:</span>}
        <input
          aria-label="Search issues"
          placeholder="Search clients, databases or ticket titles…"
          className={fieldClass}
          value={term}
          onChange={(e) => setTerm(e.target.value)}
        />
      </label>
      <Button type="submit">Search</Button>
    </>
  );
  return (
    <section className={classic ? styles.classic : ""}>
      <div className={styles.heading}>
        <div>
          {!classic && <p className={styles.eyebrow}>ORACLE · DAILY CHECKS</p>}
          <h1>{status === "closed" ? "Closed" : "Open"} Oracle issues</h1>
          {!classic && (
            <p>
              {status === "closed"
                ? "Review the history of resolved tickets."
                : "Investigate checks that need your attention."}
            </p>
          )}
        </div>
        {classic ? (
          <p className={styles.breadcrumb}>
            <span>Oracle issues</span>
            <span aria-hidden="true">/</span>
            {status === "closed" ? "Closed" : "Open"}
          </p>
        ) : (
          <Button onClick={() => setVersion((v) => v + 1)}>Refresh</Button>
        )}
      </div>
      <div className={classic ? styles.panel : undefined}>
        {classic && (
          <div className={styles.panelHeading}>
            <h2>Issues</h2>
            <Button
              aria-label="Refresh issues"
              title="Refresh issues"
              disabled={busy}
              onClick={() => setVersion((v) => v + 1)}
            >
              <svg
                viewBox="0 0 24 24"
                width="18"
                height="18"
                fill="none"
                stroke="currentColor"
                strokeWidth="1.6"
                aria-hidden="true"
              >
                <path d="M20 6v6h-6M20 12a8 8 0 1 0-2.1 5.4" />
              </svg>
            </Button>
          </div>
        )}
        <form
          className={styles.filters}
          onSubmit={(e) => {
            e.preventDefault();
            change({ q: term, page: 1 });
          }}
        >
          {!classic && searchControls}
          <select
            aria-label="Check type"
            className={fieldClass}
            value={search.check || ""}
            onChange={(e) => change({ check: e.target.value, page: 1 })}
          >
            <option value="">All check types</option>
            {Object.keys(checkLabels).map((x) => (
              <option key={x} value={x}>
                {checkLabels[x]}
              </option>
            ))}
          </select>
          <select
            aria-label="Sort issues"
            className={fieldClass}
            value={search.sort || "created_at"}
            onChange={(e) => change({ sort: e.target.value, page: 1 })}
          >
            <option value="created_at">Created date</option>
            <option value="number">Ticket number</option>
            <option value="client">Client</option>
            <option value="database">Database</option>
            <option value="check_type">Check type</option>
            <option value="assessment_type">Assessment type</option>
            {(!classic || search.sort === "status") && (
              <option value="status">Status</option>
            )}
          </select>
          <select
            aria-label="Sort direction"
            className={fieldClass}
            value={search.order || "desc"}
            onChange={(e) => change({ order: e.target.value, page: 1 })}
          >
            <option value="desc">Descending</option>
            <option value="asc">Ascending</option>
          </select>
          {classic && searchControls}
        </form>
        <details
          className={styles.moreFilters}
          onToggle={(e) => setFiltersOpen(e.currentTarget.open)}
        >
          <summary>Filter by client, database, assessment or date</summary>
          <div className={styles.filterGrid}>
            <RemoteSelect
              label="Client"
              path="/clients"
              active={filtersOpen}
              value={search.client_id || ""}
              onChange={(value) =>
                change({ client_id: value, database_id: "", page: 1 })
              }
            />
            <RemoteSelect
              label="Database"
              path="/databases"
              active={filtersOpen}
              clientId={search.client_id}
              value={search.database_id || ""}
              onChange={(value) => change({ database_id: value, page: 1 })}
            />
            <label>
              Assessment
              <select
                className={fieldClass}
                value={search.assessment_type || ""}
                onChange={(e) =>
                  change({ assessment_type: e.target.value, page: 1 })
                }
              >
                <option value="">All assessment types</option>
                <option value="daily_check">Daily check</option>
                <option value="missing">Missing</option>
                <option value="standby">Standby</option>
              </select>
            </label>
            <label>
              Ticket number
              <input
                className={fieldClass}
                type="number"
                min="1"
                value={search.number || ""}
                onChange={(e) => change({ number: e.target.value, page: 1 })}
              />
            </label>
            <label>
              Created from (UTC)
              <input
                className={fieldClass}
                type="date"
                value={search.created_from || ""}
                onChange={(e) =>
                  change({ created_from: e.target.value, page: 1 })
                }
              />
            </label>
            <label>
              Created through (UTC)
              <input
                className={fieldClass}
                type="date"
                value={search.created_to || ""}
                onChange={(e) =>
                  change({ created_to: e.target.value, page: 1 })
                }
              />
            </label>
          </div>
          <Button
            onClick={() =>
              change({
                q: "",
                check: "",
                client_id: "",
                database_id: "",
                assessment_type: "",
                number: "",
                created_from: "",
                created_to: "",
                page: 1,
              })
            }
          >
            Clear filters
          </Button>
        </details>
        <div className={styles.summary}>
          <span>{total} matching issues</span>
          <span>
            Page {page} · {rows.length} loaded
          </span>
        </div>
        {error && (
          <Notice error>
            {error}{" "}
            <Button onClick={() => setVersion((v) => v + 1)}>Retry</Button>
          </Notice>
        )}
        <TicketTable
          tickets={rows}
          appearance={appearance}
          sort={search.sort || "created_at"}
          order={search.order || "desc"}
          onSort={(sort) =>
            change({
              sort,
              order:
                search.sort === sort || (!search.sort && sort === "created_at")
                  ? search.order === "asc"
                    ? "desc"
                    : "asc"
                  : "asc",
              page: 1,
            })
          }
        />
        {!busy && !error && rows.length === 0 && (
          <Notice>No matching issues.</Notice>
        )}
        <div ref={sentinel} className={styles.sentinel}>
          {busy
            ? "Loading issues…"
            : !secondLoaded &&
                loaded === 50 &&
                (page - 1) * 100 + loaded < total
              ? "Scroll to load 50 more"
              : ""}
        </div>
        <footer className={styles.footer}>
          <small>
            New reports can change these results. Refresh to see the latest
            list.
          </small>
          <div>
            <Button
              disabled={page === 1 || busy}
              onClick={() => {
                change({ page: page - 1 });
                window.scrollTo(0, 0);
              }}
            >
              Previous
            </Button>{" "}
            <Button
              disabled={loaded < 100 || page * 100 >= total || busy}
              onClick={() => {
                change({ page: page + 1 });
                window.scrollTo(0, 0);
              }}
            >
              Next page
            </Button>
          </div>
        </footer>
      </div>
    </section>
  );
}
