import type { ButtonHTMLAttributes, ReactNode } from "react";
import styles from "./Controls.module.css";
export function Button({
  primary = false,
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { primary?: boolean }) {
  return (
    <button
      {...props}
      className={`${styles.button} ${primary ? styles.primary : ""} ${className}`}
    />
  );
}
export function Notice({
  children,
  error = false,
}: {
  children: ReactNode;
  error?: boolean;
}) {
  return (
    <div className={styles.notice} role={error ? "alert" : "status"}>
      {children}
    </div>
  );
}
export function Status({ value }: { value: string }) {
  return (
    <span className={styles.status}>
      {value === "open" ? "○" : "✓"} {value}
    </span>
  );
}
export const fieldClass = styles.field;
