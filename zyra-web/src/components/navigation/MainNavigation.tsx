import { Link, useLocation } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useTheme } from "../../contexts/ThemeContext";
import styles from "./MainNavigation.module.css";
export function MainNavigation() {
  const [active, setActive] = useState<string | null>(null);
  const nav = useRef<HTMLElement>(null);
  const trigger = useRef<HTMLButtonElement | null>(null);
  const location = useLocation();
  const { session, logout } = useAuth();
  const { dark, toggle } = useTheme();
  useEffect(() => setActive(null), [location.href]);
  useEffect(() => {
    const outside = (e: PointerEvent) => {
      if (!nav.current?.contains(e.target as Node)) setActive(null);
    };
    const escape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && active) {
        setActive(null);
        trigger.current?.focus();
      }
    };
    document.addEventListener("pointerdown", outside);
    document.addEventListener("keydown", escape);
    return () => {
      document.removeEventListener("pointerdown", outside);
      document.removeEventListener("keydown", escape);
    };
  }, [active]);
  function menu(name: string, label = name) {
    return (
      <button
        aria-label={name === "account" ? `Account: ${label}` : undefined}
        aria-expanded={active === name}
        aria-controls={active === name ? "navigation-panel" : undefined}
        onClick={(e) => {
          trigger.current = e.currentTarget;
          setActive(active === name ? null : name);
        }}
        className={active === name ? styles.active : ""}
      >
        {name === "account" ? (
          <span className={styles.avatar} aria-hidden>
            {label.trim().charAt(0).toUpperCase() || "?"}
          </span>
        ) : (
          label
        )}{" "}
        <span aria-hidden>⌄</span>
      </button>
    );
  }
  return (
    <header
      ref={nav}
      className={styles.header}
      onClick={(e) => {
        if ((e.target as HTMLElement).closest("a")) setActive(null);
      }}
    >
      <div className={styles.bar}>
        <Link to="/" className={styles.brand}>
          <span>Z</span>Zyra
        </Link>
        <nav aria-label="Main navigation" className={styles.links}>
          <Link
            to="/"
            activeProps={{ className: styles.active }}
            activeOptions={{ exact: true }}
          >
            Dashboard
          </Link>
          {menu("Issues")}
          {menu("Assessments")}
          <button disabled title="Client screens are coming later">
            Clients
          </button>
          {menu("Settings & maintenance")}
        </nav>
        <div className={styles.tools}>
          <button
            onClick={() => {
              setActive(null);
              toggle();
            }}
            aria-label={`Switch to ${dark ? "light" : "dark"} theme`}
          >
            {dark ? "☀" : "☾"}
          </button>
          {menu("account", session?.user.name || "Account")}
        </div>
      </div>
      {active && (
        <div id="navigation-panel" className={styles.panel} key={active}>
          {active === "Issues" || active === "Assessments" ? (
            <div className={styles.columns}>
              {["Oracle", "SQL"].map((engine) => (
                <section key={engine}>
                  <h3>
                    {engine}
                    {engine === "SQL" && <small> Release 1.0</small>}
                  </h3>
                  {(active === "Issues"
                    ? ["Open", "Closed"]
                    : ["Unresolved", "Resolved", "Passed", "Failed", "All"]
                  ).map((label) =>
                    engine === "Oracle" && active === "Issues" ? (
                      <Link
                        key={label}
                        to="/issues/$status"
                        params={{ status: label.toLowerCase() }}
                      >
                        {label} issues <span>→</span>
                      </Link>
                    ) : (
                      <button key={label} disabled>
                        {label}
                        {engine === "Oracle" ? " · coming later" : ""}
                      </button>
                    ),
                  )}
                </section>
              ))}
            </div>
          ) : active === "Settings & maintenance" ? (
            <p>Settings & maintenance will be added later.</p>
          ) : (
            <div className={styles.account}>
              <div>
                <strong>{session?.user.name}</strong>
                <p>
                  {session?.user.email} · {session?.user.role}
                </p>
              </div>
              <button onClick={() => void logout().catch(() => {})}>
                Sign out
              </button>
            </div>
          )}
        </div>
      )}
    </header>
  );
}
