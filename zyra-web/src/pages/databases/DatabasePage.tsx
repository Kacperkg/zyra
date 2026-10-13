import { useCallback, useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { configurationApi as api } from "../../api/configuration.api";
import { useAuth } from "../../contexts/AuthContext";
import type {
  CheckSettings,
  ClientRecord,
  DatabaseRecord,
} from "../../types/configuration";
import { optionalNumber } from "../../types/configuration-validation";
import { Button, Notice } from "../../components/ui/Controls";
import { Tabs } from "../../components/ui/Tabs";
import { ConfigurationSave } from "../../components/configuration/ConfigurationControls";
import { CheckSettingsEditor } from "../../components/configuration/CheckSettingsEditor";
import { AssessmentHistory } from "../../components/configuration/AssessmentHistory";
import { EmailSources } from "../../components/configuration/EmailSources";
import { useConfigurationGuard } from "../../components/configuration/useConfigurationGuard";
import styles from "../../components/configuration/Configuration.module.css";
const tabs = [
  { value: "notes", label: "Notes" },
  { value: "checks", label: "Checks" },
  { value: "schedules", label: "Check schedules" },
  { value: "filesystem", label: "Filesystem" },
  { value: "tablespaces", label: "Tablespaces" },
  { value: "asm_space", label: "ASM space" },
  { value: "backups", label: "Backups" },
  { value: "assessments", label: "Assessments" },
  { value: "emails", label: "Assessment emails" },
];
export function DatabasePage({ databaseId }: { databaseId: string }) {
  const { session } = useAuth();
  const [database, setDatabase] = useState<DatabaseRecord | null>(null);
  const [client, setClient] = useState<ClientRecord | null>(null);
  const [settings, setSettings] = useState<CheckSettings | null>(null);
  const [tab, setTab] = useState("notes");
  const [error, setError] = useState("");
  const [dirty, setDirty] = useState(false);
  const [busy, setBusy] = useState(false);
  const [visited, setVisited] = useState<string[]>([]);
  useConfigurationGuard(dirty, busy);
  const onDirty = useCallback((value: boolean) => setDirty(value), []);
  const edit =
    session?.user.role !== "normal" &&
    !!client &&
    !database?.archived &&
    !client.archived;
  const archive = ["admin", "owner"].includes(session?.user.role || "");
  useEffect(() => {
    const abort = new AbortController();
    setDatabase(null);
    setClient(null);
    setSettings(null);
    setError("");
    setTab("notes");
    setVisited([]);
    api
      .database(databaseId, abort.signal)
      .then(async (v) => {
        setDatabase(v);
        const c = await api.client(v.client_id, abort.signal);
        setClient(c);
      })
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [databaseId]);
  useEffect(() => {
    if (
      !["checks", "filesystem", "tablespaces", "asm_space", "backups"].includes(
        tab,
      ) ||
      settings
    )
      return;
    const abort = new AbortController();
    api
      .settings(databaseId, abort.signal)
      .then(setSettings)
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [tab, settings, databaseId]);
  useEffect(() => {
    const warn = (e: BeforeUnloadEvent) => {
      if (dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  if (!database)
    return <Notice error={!!error}>{error || "Loading database…"}</Notice>;
  return (
    <>
      <div className={styles.heading}>
        <div>
          {client && (
            <Link to="/clients/$clientId" params={{ clientId: client.id }}>
              {client.name}
            </Link>
          )}
          <h1>{database.name}</h1>
          <span className={styles.muted}>
            {database.engine === "oracle"
              ? "Oracle database"
              : "SQL instance · monitoring unavailable"}
          </span>
        </div>
        {archive && !database.archived && !client?.archived && (
          <Button
            disabled={busy}
            onClick={async () => {
              if (dirty && !window.confirm("Discard unsaved changes?")) return;
              if (
                !window.confirm(
                  `Archive ${database.name}? Historical tickets and assessments remain available.`,
                )
              )
                return;
              setBusy(true);
              try {
                await api.archiveDatabase(databaseId);
                setDatabase({ ...database, archived: true });
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            Archive database
          </Button>
        )}
      </div>
      {(database.archived || client?.archived) && (
        <Notice>
          Archived configuration. Historical records remain available; editing
          is disabled.
        </Notice>
      )}
      {error && <Notice error>{error}</Notice>}
      <dl className={`${styles.identity} ${styles.panel}`}>
        <div>
          <dt>Hostname</dt>
          <dd>{database.hostname || "Not recorded"}</dd>
        </div>
        <div>
          <dt>IP address</dt>
          <dd>{database.ip || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Latest observed Unique ID</dt>
          <dd>
            {database.observed_unique_id || "Not present in a received report"}
            {database.observed_unique_id_received_at && (
              <div className={styles.muted}>
                Observed{" "}
                {new Date(
                  database.observed_unique_id_received_at,
                ).toLocaleString()}
              </div>
            )}
          </dd>
        </div>
        <div>
          <dt>Maximum lag</dt>
          <dd>
            {database.max_lag_sequences === null
              ? "Not configured"
              : `${database.max_lag_sequences} sequences`}
            <div className={styles.muted}>
              Configuration only; evaluation deferred.
            </div>
          </dd>
        </div>
      </dl>
      <Tabs
        id="database"
        label="Database configuration"
        value={tab}
        items={tabs}
        onChange={(next) => {
          if (busy) return;
          if (dirty && !window.confirm("Discard unsaved changes in this tab?"))
            return;
          setDirty(false);
          setTab(next);
          setVisited((current) =>
            current.includes(next) ? current : [...current, next],
          );
        }}
      />
      <section
        className={styles.panel}
        id="database-panel"
        role="tabpanel"
        aria-labelledby={`database-${tab}`}
      >
        <h2>{tabs.find((item) => item.value === tab)?.label}</h2>
        {tab === "notes" ? (
          <DatabaseDetails
            key={database.id + String(database.archived)}
            database={database}
            editable={edit}
            onSaved={setDatabase}
            onDirty={onDirty}
            onBusy={setBusy}
          />
        ) : ["assessments", "emails", "schedules"].includes(
            tab,
          ) ? null : settings ? (
          <CheckSettingsEditor
            key={tab + String(edit)}
            databaseId={databaseId}
            tab={tab}
            editable={edit}
            settings={settings}
            onSaved={setSettings}
            onDirty={onDirty}
            onBusy={setBusy}
          />
        ) : (
          <Notice>Loading check settings…</Notice>
        )}
        {visited.includes("assessments") && (
          <div hidden={tab !== "assessments"}>
            <AssessmentHistory databaseId={databaseId} />
          </div>
        )}
        {visited.includes("emails") && (
          <div hidden={tab !== "emails"}>
            <EmailSources databaseId={databaseId} editable={edit} />
          </div>
        )}
        {visited.includes("schedules") && (
          <div hidden={tab !== "schedules"}>
            <EmailSources databaseId={databaseId} editable={edit} schedules />
          </div>
        )}
      </section>
    </>
  );
}
function DatabaseDetails({
  database,
  editable,
  onSaved,
  onDirty,
  onBusy,
}: {
  database: DatabaseRecord;
  editable: boolean;
  onSaved: (database: DatabaseRecord) => void;
  onDirty: (dirty: boolean) => void;
  onBusy: (busy: boolean) => void;
}) {
  const original = {
    name: database.name,
    hostname: database.hostname,
    ip: database.ip,
    notes: database.notes,
    lag:
      database.max_lag_sequences === null
        ? ""
        : String(database.max_lag_sequences),
  };
  const [draft, setDraft] = useState(original);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const dirty = JSON.stringify(draft) !== JSON.stringify(original);
  useEffect(() => {
    onDirty(dirty);
    return () => onDirty(false);
  }, [dirty, onDirty]);
  useEffect(() => {
    onBusy(busy);
    return () => onBusy(false);
  }, [busy, onBusy]);
  return (
    <form
      className={styles.form}
      onSubmit={async (e) => {
        e.preventDefault();
        setBusy(true);
        setError("");
        try {
          const v = await api.saveDatabase(database.id, {
            name: draft.name,
            hostname: draft.hostname,
            ip: draft.ip,
            notes: draft.notes,
            max_lag_sequences: optionalNumber(draft.lag, "sequences"),
          });
          onSaved(v);
        } catch (e) {
          setError((e as Error).message);
        } finally {
          setBusy(false);
        }
      }}
    >
      {error && <Notice error>{error}</Notice>}
      <label>
        Database-scoped notes
        <textarea
          readOnly={!editable || busy}
          value={draft.notes}
          onChange={(e) => setDraft({ ...draft, notes: e.target.value })}
        />
      </label>
      <label>
        Database name
        <input
          required
          maxLength={200}
          readOnly={!editable || busy}
          value={draft.name}
          onChange={(e) => setDraft({ ...draft, name: e.target.value })}
        />
      </label>
      <label>
        Hostname
        <input
          readOnly={!editable || busy}
          value={draft.hostname}
          onChange={(e) => setDraft({ ...draft, hostname: e.target.value })}
        />
      </label>
      <label>
        IP address
        <input
          readOnly={!editable || busy}
          value={draft.ip}
          onChange={(e) => setDraft({ ...draft, ip: e.target.value })}
        />
      </label>
      <label>
        Maximum allowed lag (sequences)
        <input
          type="number"
          min="0"
          max="2147483647"
          step="1"
          readOnly={!editable || busy}
          value={draft.lag}
          onChange={(e) => setDraft({ ...draft, lag: e.target.value })}
        />
        <small className={styles.muted}>
          Leave empty for unset. This does not enable standby evaluation.
        </small>
      </label>
      {editable && (
        <ConfigurationSave
          busy={busy}
          dirty={dirty}
          onCancel={() => setDraft(original)}
        />
      )}
    </form>
  );
}
