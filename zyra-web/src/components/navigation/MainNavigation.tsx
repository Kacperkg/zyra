import { Link, useLocation, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { useAuth } from "../../contexts/AuthContext";
import { useTheme } from "../../contexts/ThemeContext";
import { Avatar } from "../ui/Avatar";
import { notificationsApi } from "../../api/notifications.api";
import type { NotificationSummary, Page } from "../../types/api";
import styles from "./MainNavigation.module.css";
export function MainNavigation() {
  const [active, setActive] = useState<string | null>(null);
  const nav = useRef<HTMLElement>(null);
  const trigger = useRef<HTMLButtonElement | null>(null);
  const location = useLocation();
  const navigate = useNavigate();
  const { session, logout } = useAuth();
  const { dark, toggle, busy: themeBusy, error: themeError } = useTheme();
  const [unread, setUnread] = useState(0);
  const [notifications, setNotifications] =
    useState<Page<NotificationSummary> | null>(null);
  const [notificationPage, setNotificationPage] = useState(1);
  const [notificationError, setNotificationError] = useState("");
  const [notificationBusy, setNotificationBusy] = useState(false);
  const [notificationReload, setNotificationReload] = useState(0);
  const notificationGeneration = useRef(0);
  useEffect(() => {
    setUnread(0);
    setNotifications(null);
    setNotificationError("");
    setNotificationPage(1);
  }, [session?.user.id]);
  useEffect(() => {
    if (!session) return;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    const generation = ++notificationGeneration.current;
    async function poll() {
      try {
        const [countResult, listResult] = await Promise.allSettled([
          notificationsApi.unread(controller.signal),
          active === "notifications"
            ? notificationsApi.list(notificationPage, controller.signal)
            : Promise.resolve(null),
        ]);
        if (countResult.status === "rejected") throw countResult.reason;
        if (listResult.status === "rejected") throw listResult.reason;
        const count = countResult.value;
        const list = listResult.value;
        if (
          !controller.signal.aborted &&
          notificationGeneration.current === generation
        ) {
          setUnread(count.count);
          if (list) setNotifications(list);
          setNotificationError("");
        }
      } catch (e) {
        if (
          !controller.signal.aborted &&
          notificationGeneration.current === generation
        )
          setNotificationError((e as Error).message);
      } finally {
        if (!controller.signal.aborted)
          timer = setTimeout(() => void poll(), 30000);
      }
    }
    void poll();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [session?.user.id, active, notificationPage, notificationReload]);
  useEffect(() => {
    if (active === "notifications") setNotifications(null);
  }, [active, notificationPage]);
  async function readNotification(item: NotificationSummary) {
    setNotificationBusy(true);
    notificationGeneration.current++;
    try {
      await notificationsApi.read(item.id);
      setActive(null);
      await navigate({
        to: "/tickets/$ticketId",
        params: { ticketId: item.ticket_id },
      });
    } catch (e) {
      setNotificationError((e as Error).message);
    } finally {
      setNotificationBusy(false);
      setNotificationReload((x) => x + 1);
    }
  }
  async function readAll() {
    setNotificationBusy(true);
    notificationGeneration.current++;
    try {
      await notificationsApi.readAll();
    } catch (e) {
      setNotificationError((e as Error).message);
    } finally {
      setNotificationBusy(false);
      setNotificationReload((x) => x + 1);
    }
  }
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
          <Avatar user={session?.user || { name: label }} />
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
            disabled={themeBusy}
            onClick={() => {
              setActive(null);
              toggle();
            }}
            aria-label={`Switch to ${dark ? "light" : "dark"} theme`}
          >
            {dark ? "☀" : "☾"}
          </button>
          <button
            aria-label={`Notifications${unread ? `, ${unread} unread` : ""}`}
            aria-expanded={active === "notifications"}
            aria-controls={
              active === "notifications" ? "navigation-panel" : undefined
            }
            onClick={(e) => {
              trigger.current = e.currentTarget;
              setActive(active === "notifications" ? null : "notifications");
              setNotificationPage(1);
            }}
            className={styles.bell}
          >
            <svg
              width="20"
              height="20"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.7"
              aria-hidden="true"
            >
              <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" />
            </svg>
            {unread > 0 && (
              <span className={styles.count}>
                {unread > 99 ? "99+" : unread}
              </span>
            )}
          </button>
          {menu("account", session?.user.name || "Account")}
        </div>
      </div>
      {themeError && (
        <p className={styles.error} role="alert">
          {themeError}
        </p>
      )}
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
          ) : active === "notifications" ? (
            <section
              className={styles.notifications}
              aria-label="Notifications"
            >
              <div className={styles.notificationHeading}>
                <h3>Notifications</h3>
                <button
                  disabled={!unread || notificationBusy}
                  onClick={() => void readAll()}
                >
                  Mark all as read
                </button>
              </div>
              {notificationError && <p role="alert">{notificationError}</p>}
              {!notifications ? (
                <p role="status">Loading notifications…</p>
              ) : notifications.items.length ? (
                <ul>
                  {notifications.items.map((item) => (
                    <li key={item.id}>
                      <button
                        disabled={notificationBusy}
                        className={!item.read_at ? styles.unread : ""}
                        onClick={() => void readNotification(item)}
                      >
                        <Avatar
                          user={{
                            name: item.actor_name,
                            avatar_url: item.actor_avatar_url,
                          }}
                        />
                        <span>
                          <strong>{item.actor_name}</strong> mentioned you in{" "}
                          <strong>#{item.ticket_number}</strong>
                          <span className={styles.notificationTitle}>
                            {item.ticket_title}
                          </span>
                          <small>
                            {new Date(item.created_at).toLocaleString()} ·{" "}
                            {item.read_at ? "Read" : "Unread"}
                          </small>
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              ) : (
                <p>No notifications yet.</p>
              )}
              {notifications && notifications.total > 20 && (
                <div className={styles.notificationPaging}>
                  <button
                    disabled={notificationPage === 1 || notificationBusy}
                    onClick={() => setNotificationPage((x) => x - 1)}
                  >
                    Previous
                  </button>
                  <span>Page {notificationPage}</span>
                  <button
                    disabled={
                      notificationPage * 20 >= notifications.total ||
                      notificationBusy
                    }
                    onClick={() => setNotificationPage((x) => x + 1)}
                  >
                    Next
                  </button>
                </div>
              )}
            </section>
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
              <Link to="/profile">Profile & saved tickets</Link>
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
