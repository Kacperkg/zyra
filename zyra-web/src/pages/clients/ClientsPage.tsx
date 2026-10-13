import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { configurationApi as api } from "../../api/configuration.api";
import { useAuth } from "../../contexts/AuthContext";
import type { Page } from "../../types/api";
import type { ClientRecord } from "../../types/configuration";
import { Button, Notice } from "../../components/ui/Controls";
import { ListPagination } from "../../components/configuration/ConfigurationControls";
import styles from "../../components/configuration/Configuration.module.css";
export function ClientsPage() {
  const { session } = useAuth();
  const edit = session?.user.role !== "normal";
  const [data, setData] = useState<Page<ClientRecord> | null>(null);
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState("");
  const [error, setError] = useState("");
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [reload, setReload] = useState(0);
  useEffect(() => {
    const abort = new AbortController();
    setData(null);
    const timer = setTimeout(() => {
      api
        .clients(page, query, abort.signal)
        .then(setData)
        .catch((e) => {
          if (!abort.signal.aborted) setError(e.message);
        });
    }, 200);
    return () => {
      abort.abort();
      clearTimeout(timer);
    };
  }, [page, query, reload]);
  return (
    <>
      <div className={styles.heading}>
        <div>
          <h1>Clients</h1>
          <p className={styles.muted}>Configured database estates.</p>
        </div>
      </div>
      {error && <Notice error>{error}</Notice>}
      <div className={styles.panel}>
        <label className={styles.form}>
          Search clients
          <input
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </label>
        {!data ? (
          <Notice>Loading clients…</Notice>
        ) : (
          <>
            <ul className={styles.list}>
              {data.items.map((client) => (
                <li key={client.id}>
                  <Link
                    to="/clients/$clientId"
                    params={{ clientId: client.id }}
                  >
                    {client.name}
                  </Link>
                </li>
              ))}
            </ul>
            {!data.items.length && <p>No clients found.</p>}
            <ListPagination
              page={page}
              total={data.total}
              count={data.items.length}
              onChange={setPage}
            />
          </>
        )}
      </div>
      {edit && (
        <section className={styles.panel}>
          <h2>Add client</h2>
          <form
            className={styles.form}
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api.saveClient("", { name });
                setName("");
                setPage(1);
                setReload((x) => x + 1);
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <label>
              Client name
              <input
                required
                maxLength={200}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <div>
              <Button primary disabled={busy}>
                {busy ? "Adding…" : "Add client"}
              </Button>
            </div>
          </form>
        </section>
      )}
    </>
  );
}
