import { useEffect, useRef, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import { configurationApi as api } from "../../api/configuration.api";
import { useAuth } from "../../contexts/AuthContext";
import type {
  DatabaseRecord,
  EmailSource,
  SourceSchedule,
} from "../../types/configuration";
import { Notice } from "../../components/ui/Controls";
import { ConfigurationSave } from "../../components/configuration/ConfigurationControls";
import { useConfigurationGuard } from "../../components/configuration/useConfigurationGuard";
import styles from "../../components/configuration/Configuration.module.css";
const days = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
const blank = {
  subject: "",
  sender: "",
  mailbox: "",
  assessment_type: "daily_check" as EmailSource["assessment_type"],
  timezone: "Europe/London",
  enabled: false,
  schedules: [] as SourceSchedule[],
  weekdays: [...days],
};
function editableSource(source: EmailSource) {
  return {
    subject: source.subject,
    sender: source.sender,
    mailbox: source.mailbox,
    assessment_type: source.assessment_type,
    timezone: source.timezone,
    enabled: source.enabled,
    schedules: source.schedules || [],
    weekdays: source.weekdays || [],
  };
}
export function EmailSourcePage({
  databaseId,
  sourceId,
}: {
  databaseId: string;
  sourceId: string;
}) {
  const { session } = useAuth();
  const navigate = useNavigate();
  const [database, setDatabase] = useState<DatabaseRecord | null>(null);
  const [editable, setEditable] = useState(false);
  const [source, setSource] = useState<EmailSource | null>(null);
  const [draft, setDraft] = useState(blank);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [success, setSuccess] = useState("");
  const savedNavigation = useRef(false);
  useEffect(() => {
    const abort = new AbortController();
    setReady(false);
    setError("");
    Promise.all([
      api.database(databaseId, abort.signal),
      sourceId === "new"
        ? Promise.resolve(null)
        : api.source(sourceId, abort.signal),
    ])
      .then(async ([db, row]) => {
        if (row && row.database_id !== databaseId)
          throw new Error(
            "Email configuration does not belong to this database.",
          );
        const client = await api.client(db.client_id, abort.signal);
        setDatabase(db);
        setEditable(
          session?.user.role !== "normal" && !db.archived && !client.archived,
        );
        setSource(row);
        setDraft(
          row
            ? editableSource(row)
            : { ...blank, schedules: [], weekdays: [...days] },
        );
        setReady(true);
      })
      .catch((e) => {
        if (!abort.signal.aborted) setError(e.message);
      });
    return () => abort.abort();
  }, [databaseId, sourceId, session?.user.role]);
  const dirty =
    sourceId === "new"
      ? JSON.stringify(draft) !== JSON.stringify(blank)
      : !!source &&
        JSON.stringify(draft) !== JSON.stringify(editableSource(source));
  useConfigurationGuard(dirty, busy, () => savedNavigation.current);
  function updateWindow(name: string, field: "start" | "end", value: string) {
    setDraft({
      ...draft,
      schedules: draft.schedules.map((w) =>
        w.name === name ? { ...w, [field]: value } : w,
      ),
    });
  }
  return (
    <>
      <div className={styles.heading}>
        <div>
          <Link to="/databases/$databaseId" params={{ databaseId }}>
            {database?.name || "Database"}
          </Link>
          <h1>
            {sourceId === "new"
              ? "Add assessment email"
              : source?.subject || "Assessment email"}
          </h1>
        </div>
      </div>
      {error && <Notice error>{error}</Notice>}
      {!ready ? (
        !error && <Notice>Loading assessment email…</Notice>
      ) : (
        <section className={styles.panel}>
          <Notice>
            These settings describe expected mail only. Live mailbox ingestion
            and background monitoring are not connected. Standby processing is
            deferred.
          </Notice>
          {success && <Notice>{success}</Notice>}
          <form
            className={styles.form}
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              setSuccess("");
              try {
                const saved = await api.saveSource(
                  databaseId,
                  sourceId === "new" ? "" : sourceId,
                  draft,
                );
                setSource(saved);
                setDraft(editableSource(saved));
                setSuccess("Assessment email configuration saved.");
                if (sourceId === "new") {
                  savedNavigation.current = true;
                  setBusy(false);
                  await navigate({
                    to: "/databases/$databaseId/emails/$sourceId",
                    params: { databaseId, sourceId: saved.id },
                  });
                  savedNavigation.current = false;
                }
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <fieldset disabled={!editable || busy} className={styles.form}>
              <legend>Details</legend>
              <label>
                Subject
                <input
                  required
                  maxLength={500}
                  value={draft.subject}
                  onChange={(e) =>
                    setDraft({ ...draft, subject: e.target.value })
                  }
                />
                <small className={styles.muted}>
                  Current manual processing uses exact subject/sender matching.
                  Migration and mailbox-aware matching are not implemented.
                </small>
              </label>
              <label>
                Sender address
                <input
                  type="email"
                  required
                  value={draft.sender}
                  onChange={(e) =>
                    setDraft({ ...draft, sender: e.target.value })
                  }
                />
              </label>
              <label>
                Receiving mailbox
                <input
                  type="email"
                  value={draft.mailbox}
                  onChange={(e) =>
                    setDraft({ ...draft, mailbox: e.target.value })
                  }
                />
              </label>
              <label>
                Assessment type
                <select
                  value={draft.assessment_type}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      assessment_type: e.target
                        .value as EmailSource["assessment_type"],
                    })
                  }
                >
                  <option value="daily_check">Daily check</option>
                  <option value="standby_check">
                    Standby check (processing deferred)
                  </option>
                </select>
              </label>
              <label>
                Active
                <select
                  value={draft.enabled ? "yes" : "no"}
                  onChange={(e) =>
                    setDraft({ ...draft, enabled: e.target.value === "yes" })
                  }
                >
                  <option value="yes">Yes</option>
                  <option value="no">No</option>
                </select>
              </label>
            </fieldset>
            <fieldset disabled={!editable || busy} className={styles.form}>
              <legend>Expected delivery windows</legend>
              <label>
                Timezone
                <input
                  required
                  value={draft.timezone}
                  onChange={(e) =>
                    setDraft({ ...draft, timezone: e.target.value })
                  }
                />
                <small className={styles.muted}>
                  IANA timezone, for example Europe/London. These are not
                  browser-local times.
                </small>
              </label>
              <div className={styles.checks}>
                {days.map((day) => (
                  <label key={day} className={styles.check}>
                    <input
                      type="checkbox"
                      checked={draft.weekdays.includes(day)}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          weekdays: e.target.checked
                            ? days.filter(
                                (d) => d === day || draft.weekdays.includes(d),
                              )
                            : draft.weekdays.filter((d) => d !== day),
                        })
                      }
                    />
                    {day.charAt(0).toUpperCase() + day.slice(1)}
                  </label>
                ))}
              </div>
              {["AM", "PM"].map((name) => {
                const window = draft.schedules.find((w) => w.name === name);
                return (
                  <fieldset key={name} className={styles.form}>
                    <legend>
                      {name === "AM" ? "Morning (AM)" : "Evening (PM)"}
                    </legend>
                    <label className={styles.check}>
                      <input
                        type="checkbox"
                        checked={!!window}
                        onChange={(e) =>
                          setDraft({
                            ...draft,
                            schedules: e.target.checked
                              ? [
                                  ...draft.schedules,
                                  { name, start: "", end: "" },
                                ]
                              : draft.schedules.filter((w) => w.name !== name),
                          })
                        }
                      />
                      Expect this email
                    </label>
                    {window && (
                      <>
                        <label>
                          Between
                          <input
                            type="time"
                            required
                            value={window.start}
                            onChange={(e) =>
                              updateWindow(name, "start", e.target.value)
                            }
                          />
                        </label>
                        <label>
                          And
                          <input
                            type="time"
                            required
                            value={window.end}
                            onChange={(e) =>
                              updateWindow(name, "end", e.target.value)
                            }
                          />
                        </label>
                      </>
                    )}
                  </fieldset>
                );
              })}
              {draft.schedules
                .filter((w) => !["AM", "PM"].includes(w.name))
                .map((window) => (
                  <div key={window.name}>
                    <strong>Existing window: {window.name}</strong>
                    <label>
                      Start
                      <input
                        type="time"
                        value={window.start}
                        onChange={(e) =>
                          updateWindow(window.name, "start", e.target.value)
                        }
                      />
                    </label>
                    <label>
                      End
                      <input
                        type="time"
                        value={window.end}
                        onChange={(e) =>
                          updateWindow(window.name, "end", e.target.value)
                        }
                      />
                    </label>
                  </div>
                ))}
              <p className={styles.muted}>
                The same windows apply to each selected weekday. Overnight
                windows are allowed; overlaps are rejected. Saved changes apply
                going forward, not to elapsed windows. A received incomplete
                report is not counted as an absent email.
              </p>
            </fieldset>
            {source?.effective_at && (
              <small className={styles.muted}>
                Schedule effective from{" "}
                {new Date(source.effective_at).toLocaleString()}
              </small>
            )}
            {editable && (
              <ConfigurationSave
                busy={busy}
                dirty={dirty}
                onCancel={() =>
                  setDraft(source ? editableSource(source) : blank)
                }
              />
            )}
            {!editable && (
              <p className={styles.muted}>
                View only. Trusted users and above can edit active database
                configurations.
              </p>
            )}
          </form>
        </section>
      )}
    </>
  );
}
