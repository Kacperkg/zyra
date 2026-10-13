import { useEffect, useState } from "react";
import type { CheckSettings } from "../../types/configuration";
import { checkLabels } from "../../types/ticket-search";
import { configurationApi as api } from "../../api/configuration.api";
import { resourceRule } from "../../types/configuration-validation";
import { Button, Notice } from "../ui/Controls";
import { ConfigurationSave } from "./ConfigurationControls";
import styles from "./Configuration.module.css";
export function CheckSettingsEditor({
  databaseId,
  tab,
  editable,
  settings,
  onSaved,
  onDirty,
  onBusy,
}: {
  databaseId: string;
  tab: string;
  editable: boolean;
  settings: CheckSettings;
  onSaved: (settings: CheckSettings) => void;
  onDirty: (dirty: boolean) => void;
  onBusy: (busy: boolean) => void;
}) {
  const [draft, setDraft] = useState(settings);
  const [name, setName] = useState("");
  const [threshold, setThreshold] = useState(tab === "filesystem" ? "90" : "");
  const [ignore, setIgnore] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const dirty =
    JSON.stringify(settings) !== JSON.stringify(draft) ||
    !!name ||
    threshold !== (tab === "filesystem" ? "90" : "") ||
    ignore;
  useEffect(() => {
    onDirty(dirty);
    return () => onDirty(false);
  }, [dirty, onDirty]);
  useEffect(() => {
    onBusy(busy);
    return () => onBusy(false);
  }, [busy, onBusy]);
  const key = tab === "tablespaces" ? "tablespace" : tab;
  const kind =
    key === "filesystem"
      ? "max_used_percent"
      : ["tablespace", "asm_space"].includes(key)
        ? "min_free_percent"
        : null;
  const rules = draft.resources?.[key] || {};
  function reset() {
    setDraft(settings);
    setName("");
    setThreshold(tab === "filesystem" ? "90" : "");
    setIgnore(false);
    setError("");
  }
  return (
    <form
      onSubmit={async (e) => {
        e.preventDefault();
        if (
          name ||
          threshold !== (tab === "filesystem" ? "90" : "") ||
          ignore
        ) {
          setError("Add or clear the new resource before saving.");
          return;
        }
        setBusy(true);
        setError("");
        try {
          const saved = await api.saveSettings(databaseId, draft);
          setDraft(saved);
          onSaved(saved);
        } catch (e) {
          setError((e as Error).message);
        } finally {
          setBusy(false);
        }
      }}
    >
      {error && <Notice error>{error}</Notice>}
      {tab === "checks" ? (
        <>
          <div className={styles.checks}>
            {Object.entries(checkLabels).map(([key, label]) => (
              <label className={styles.check} key={key}>
                <input
                  type="checkbox"
                  disabled={!editable || busy}
                  checked={!!draft.selected?.[key]}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      selected: { ...draft.selected, [key]: e.target.checked },
                    })
                  }
                />
                {label}
              </label>
            ))}
          </div>
          <p className={styles.muted}>
            Selected checks apply to future assessments. Sequence-lag evaluation
            is not implemented yet.
          </p>
        </>
      ) : (
        <>
          <p className={styles.muted}>
            {kind === "max_used_percent"
              ? "Maximum used percentage. New filesystem rules start at 90%; only explicitly configured resources with thresholds are evaluated."
              : kind
                ? "Minimum free percentage. An empty threshold is unset, not zero."
                : "Backup resource ignore rules. No numeric backup threshold."}{" "}
            Resource names match the report exactly; filesystem names may be
            disk names or Linux mounts.
          </p>
          <div className={styles.tableWrap}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Resource</th>
                  <th>Ignore</th>
                  {kind && (
                    <th>
                      {kind === "max_used_percent"
                        ? "Maximum used %"
                        : "Minimum free %"}
                    </th>
                  )}
                  <th />
                </tr>
              </thead>
              <tbody>
                {Object.entries(rules).map(([resource, rule]) => (
                  <tr key={resource}>
                    <td>{resource}</td>
                    <td>
                      <input
                        aria-label={`Ignore ${resource}`}
                        type="checkbox"
                        disabled={!editable || busy}
                        checked={rule.ignore}
                        onChange={(e) =>
                          setDraft({
                            ...draft,
                            resources: {
                              ...draft.resources,
                              [key]: {
                                ...rules,
                                [resource]: {
                                  ...rule,
                                  ignore: e.target.checked,
                                },
                              },
                            },
                          })
                        }
                      />
                    </td>
                    {kind && (
                      <td>
                        <input
                          aria-label={`Threshold for ${resource}`}
                          type="number"
                          min="0"
                          max="100"
                          step="any"
                          disabled={!editable || busy}
                          value={rule[kind] ?? ""}
                          onChange={(e) => {
                            const next = { ...rule };
                            if (e.target.value === "") delete next[kind];
                            else next[kind] = Number(e.target.value);
                            setDraft({
                              ...draft,
                              resources: {
                                ...draft.resources,
                                [key]: { ...rules, [resource]: next },
                              },
                            });
                          }}
                        />
                      </td>
                    )}
                    <td>
                      {editable && (
                        <Button
                          type="button"
                          disabled={busy}
                          onClick={() => {
                            const next = { ...rules };
                            delete next[resource];
                            setDraft({
                              ...draft,
                              resources: { ...draft.resources, [key]: next },
                            });
                          }}
                        >
                          Remove override
                        </Button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!Object.keys(rules).length && (
            <p>No resource overrides configured.</p>
          )}
          {editable && (
            <fieldset className={styles.form} disabled={busy}>
              <legend>Add resource override</legend>
              <label>
                Exact resource name
                <input value={name} onChange={(e) => setName(e.target.value)} />
              </label>
              {kind && (
                <label>
                  {kind === "max_used_percent"
                    ? "Maximum used %"
                    : "Minimum free %"}
                  <input
                    type="number"
                    min="0"
                    max="100"
                    step="any"
                    value={threshold}
                    onChange={(e) => setThreshold(e.target.value)}
                  />
                </label>
              )}
              <label className={styles.check}>
                <input
                  type="checkbox"
                  checked={ignore}
                  onChange={(e) => setIgnore(e.target.checked)}
                />
                Ignore this resource
              </label>
              <div>
                <Button
                  type="button"
                  onClick={() => {
                    try {
                      if (!name.trim())
                        throw new Error("Resource name is required.");
                      if (name in rules)
                        throw new Error(
                          "This resource already has an override.",
                        );
                      setDraft({
                        ...draft,
                        resources: {
                          ...draft.resources,
                          [key]: {
                            ...rules,
                            [name]: resourceRule(ignore, threshold, kind),
                          },
                        },
                      });
                      setName("");
                      setThreshold(tab === "filesystem" ? "90" : "");
                      setIgnore(false);
                      setError("");
                    } catch (e) {
                      setError((e as Error).message);
                    }
                  }}
                >
                  Add override
                </Button>
              </div>
            </fieldset>
          )}
        </>
      )}
      {editable && (
        <ConfigurationSave busy={busy} dirty={dirty} onCancel={reset} />
      )}
    </form>
  );
}
