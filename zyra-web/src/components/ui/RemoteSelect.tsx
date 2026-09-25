import { useEffect, useState } from "react";
import { request } from "../../api/client";
import type { Page } from "../../types/api";
import { fieldClass } from "./Controls";
import styles from "./RemoteSelect.module.css";
/** Bounded, server-searched choices; never preloads an entire catalogue. */
export function RemoteSelect({
  label,
  path,
  value,
  onChange,
  active,
  clientId,
}: {
  label: string;
  path: string;
  value: string;
  onChange: (value: string) => void;
  active: boolean;
  clientId?: string;
}) {
  const [term, setTerm] = useState("");
  const [items, setItems] = useState<{ id: string; name: string }[]>([]);
  const [error, setError] = useState("");
  const [total, setTotal] = useState(0);
  useEffect(() => {
    if (!active) return;
    const controller = new AbortController();
    const timer = setTimeout(() => {
      setError("");
      const query = new URLSearchParams({
        limit: "50",
        page: "1",
        q: term,
        sort: "name",
        order: "asc",
      });
      if (clientId) query.set("client_id", clientId);
      request<Page<{ id: string; name: string }>>(`${path}?${query}`, {
        signal: controller.signal,
      })
        .then((data) => {
          if (!controller.signal.aborted) {
            setItems(data.items);
            setTotal(data.total);
          }
        })
        .catch((e) => {
          if (!controller.signal.aborted) setError(e.message);
        });
    }, 250);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [path, term, active, clientId]);
  return (
    <div className={styles.field}>
      <label>
        {label}
        <input
          aria-label={`Search ${label.toLowerCase()} choices`}
          className={fieldClass}
          placeholder={`Search ${label.toLowerCase()}…`}
          value={term}
          onChange={(e) => setTerm(e.target.value)}
        />
        <select
          aria-label={label}
          className={fieldClass}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        >
          <option value="">All {label.toLowerCase()}s</option>
          {value && !items.some((item) => item.id === value) && (
            <option value={value}>
              Selected {label.toLowerCase()} ({value})
            </option>
          )}
          {items.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </select>
      </label>
      {total > 50 && (
        <small>First 50 of {total}. Search to narrow choices.</small>
      )}
      {error && <small role="alert">{error}</small>}
    </div>
  );
}
