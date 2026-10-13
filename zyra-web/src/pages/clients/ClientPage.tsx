import { useEffect, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import { configurationApi as api } from "../../api/configuration.api";
import { useAuth } from "../../contexts/AuthContext";
import type { Page } from "../../types/api";
import type { ClientRecord, DatabaseRecord } from "../../types/configuration";
import { Button, Notice } from "../../components/ui/Controls";
import {
  ConfigurationSave,
  ListPagination,
} from "../../components/configuration/ConfigurationControls";
import styles from "../../components/configuration/Configuration.module.css";
import { useConfigurationGuard } from "../../components/configuration/useConfigurationGuard";
import { useRef } from "react";
export function ClientPage({ clientId }: { clientId: string }) {
  const { session } = useAuth();
  const navigate = useNavigate();
  const edit = session?.user.role !== "normal";
  const archive = ["admin", "owner"].includes(session?.user.role || "");
  const [client, setClient] = useState<ClientRecord | null>(null);
  const [draft, setDraft] = useState({ name: "", notes: "" });
  const [data, setData] = useState<Page<DatabaseRecord> | null>(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [databaseName, setDatabaseName] = useState("");
  const [reload, setReload] = useState(0);
  const savedNavigation = useRef(false);
  const dirty =
    !!client && (draft.name !== client.name || draft.notes !== client.notes);
  useConfigurationGuard(
    dirty || !!databaseName,
    busy,
    () => savedNavigation.current,
  );
  useEffect(() => {
    const abort = new AbortController();
    setClient(null);
    setError("");
    api
      .client(clientId, abort.signal)
      .then((v) => {
        setClient(v);
        setDraft({ name: v.name, notes: v.notes });
      })
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [clientId]);
  useEffect(() => {
    const abort = new AbortController();
    setData(null);
    api
      .databases(clientId, page, abort.signal)
      .then(setData)
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [clientId, page, reload]);
  if (!client)
    return <Notice error={!!error}>{error || "Loading client…"}</Notice>;
  return (
    <>
      <div className={styles.heading}>
        <div>
          <Link to="/clients">Clients</Link>
          <h1>{client.name}</h1>
          {client.archived && (
            <Notice>
              Archived client. Historical records remain available.
            </Notice>
          )}
        </div>
        {archive && !client.archived && (
          <Button
            disabled={busy}
            onClick={async () => {
              if (
                !window.confirm(
                  `Archive ${client.name}? Historical tickets and assessments remain available.`,
                )
              )
                return;
              setBusy(true);
              try {
                await api.archiveClient(clientId);
                setClient({ ...client, archived: true });
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            Archive client
          </Button>
        )}
      </div>
      {error && <Notice error>{error}</Notice>}
      <div className={styles.columns}>
        <section className={styles.panel}>
          <h2>SQL instances</h2>
          <p className={styles.muted}>
            SQL monitoring is planned for Release 1.0 and is not available in
            this proof of concept.
          </p>
        </section>
        <section className={styles.panel}>
          <h2>Oracle databases</h2>
          {!data ? (
            <Notice>Loading databases…</Notice>
          ) : (
            <>
              <ul className={styles.list}>
                {data.items.map((db) => (
                  <li key={db.id}>
                    <Link
                      to="/databases/$databaseId"
                      params={{ databaseId: db.id }}
                    >
                      {db.name}
                    </Link>
                    <div className={styles.muted}>
                      {db.hostname || "Hostname not configured"}
                    </div>
                  </li>
                ))}
              </ul>
              {!data.items.length && <p>No Oracle databases configured.</p>}
              <ListPagination
                page={page}
                total={data.total}
                count={data.items.length}
                onChange={setPage}
              />
            </>
          )}
          {edit && !client.archived && (
            <form
              className={styles.form}
              onSubmit={async (e) => {
                e.preventDefault();
                setBusy(true);
                setError("");
                try {
                  const db = await api.saveDatabase("", {
                    name: databaseName,
                    client_id: clientId,
                    engine: "oracle",
                  });
                  savedNavigation.current = true;
                  await navigate({
                    to: "/databases/$databaseId",
                    params: { databaseId: db.id },
                  });
                  savedNavigation.current = false;
                } catch (e) {
                  setError((e as Error).message);
                } finally {
                  setBusy(false);
                }
              }}
            >
              <label>
                New database name
                <input
                  required
                  maxLength={200}
                  value={databaseName}
                  onChange={(e) => setDatabaseName(e.target.value)}
                />
              </label>
              <div>
                <Button primary disabled={busy}>
                  Add Oracle database
                </Button>
              </div>
            </form>
          )}
        </section>
      </div>
      <section className={styles.panel}>
        <h2>Client details</h2>
        <form
          className={styles.form}
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            setError("");
            try {
              const v = await api.saveClient(clientId, draft);
              setClient(v);
            } catch (e) {
              setError((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <label>
            Name
            <input
              required
              maxLength={200}
              readOnly={!edit || client.archived}
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
            />
          </label>
          <label>
            Client notes
            <textarea
              readOnly={!edit || client.archived}
              value={draft.notes}
              onChange={(e) => setDraft({ ...draft, notes: e.target.value })}
            />
          </label>
          {edit && !client.archived && (
            <ConfigurationSave
              busy={busy}
              dirty={dirty}
              onCancel={() =>
                setDraft({ name: client.name, notes: client.notes })
              }
            />
          )}
        </form>
      </section>
    </>
  );
}
