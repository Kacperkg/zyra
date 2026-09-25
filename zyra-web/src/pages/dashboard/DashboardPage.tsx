import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { request } from "../../api/client";
import { Notice, Button } from "../../components/ui/Controls";
import styles from "./DashboardPage.module.css";
export function DashboardPage() {
  const [count, setCount] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [refresh, setRefresh] = useState(0);
  useEffect(() => {
    let alive = true;
    setError("");
    request<{ oracle: { open_issues: number } }>("/dashboard")
      .then((data) => {
        if (alive) setCount(data.oracle.open_issues);
      })
      .catch((e) => {
        if (alive) setError(e.message);
      });
    return () => {
      alive = false;
    };
  }, [refresh]);
  return (
    <>
      <div className={styles.heading}>
        <div>
          <p className={styles.eyebrow}>DATABASE HEALTH</p>
          <h1>Dashboard</h1>
          <p>Everything that needs your attention.</p>
        </div>
        <Button onClick={() => setRefresh((x) => x + 1)}>Refresh</Button>
      </div>
      {error && <Notice error>{error}</Notice>}
      <div className={styles.columns}>
        <section className={styles.card}>
          <div className={styles.cardTitle}>
            <h2>Oracle</h2>
            <span>Daily checks</span>
          </div>
          <strong className={styles.count}>{count ?? "—"}</strong>
          <p>Open Oracle issues</p>
          <div className={styles.actions}>
            <Link to="/issues/$status" params={{ status: "open" }}>
              Open issues <span>→</span>
            </Link>
            <Link to="/issues/$status" params={{ status: "closed" }}>
              Closed issues <span>→</span>
            </Link>
          </div>
        </section>
        <section className={styles.card}>
          <div className={styles.cardTitle}>
            <h2>SQL</h2>
            <span>Release 1.0</span>
          </div>
          <strong className={styles.count}>—</strong>
          <p>SQL monitoring is not active yet.</p>
          <div className={styles.actions}>
            <Button disabled>Open issues</Button>
            <Button disabled>Closed issues</Button>
          </div>
        </section>
      </div>
    </>
  );
}
