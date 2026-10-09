import styles from "./StatusBadge.module.css";
export function StatusBadge({ status }: { status: "open" | "closed" }) {
  return (
    <span className={`${styles.badge} ${styles[status]}`}>
      <span aria-hidden>{status === "open" ? "●" : "✓"}</span>{" "}
      {status === "open" ? "Open" : "Closed"}
    </span>
  );
}
