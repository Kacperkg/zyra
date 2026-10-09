import { useEffect, useState, type FormEvent } from "react";
import { Link } from "@tanstack/react-router";
import { useAuth } from "../../contexts/AuthContext";
import { useTheme } from "../../contexts/ThemeContext";
import { usersApi } from "../../api/users.api";
import {
  listSavedTickets,
  saveTicket,
  unsaveTicket,
} from "../../api/saved-tickets.api";
import type { Page, SavedTicketSummary } from "../../types/api";
import { Avatar } from "../../components/ui/Avatar";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { Button, Notice, fieldClass } from "../../components/ui/Controls";
import styles from "./ProfilePage.module.css";
import {
  clipboardImageSources,
  validateImageURL,
} from "../../components/editor/clipboard";
export function ProfilePage() {
  const { session, updateUser } = useAuth();
  const {
    dark,
    appearance,
    setTheme,
    setAppearance,
    busy: preferencesBusy,
    error: preferencesError,
  } = useTheme();
  const [name, setName] = useState(session?.user.name || "");
  const [avatar, setAvatar] = useState(session?.user.avatar_url || "");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Page<SavedTicketSummary> | null>(null);
  const [loading, setLoading] = useState(true);
  const [reload, setReload] = useState(0);
  const [editing, setEditing] = useState("");
  const [title, setTitle] = useState("");
  const [mutation, setMutation] = useState("");
  useEffect(() => {
    const timeout = setTimeout(() => {
      setSearch(query);
      setPage(1);
    }, 250);
    return () => clearTimeout(timeout);
  }, [query]);
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    listSavedTickets(search, page, controller.signal)
      .then((value) => {
        if (controller.signal.aborted) return;
        setData(value);
        if (!value.items.length && page > 1) setPage(page - 1);
      })
      .catch((e) => {
        if (!controller.signal.aborted) setError(e.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [search, page, reload, session?.user.id]);
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const imageURL = avatar.trim();
      await validateImageURL(imageURL);
      const user = await usersApi.update({
        name: name.trim(),
        avatar_url: imageURL,
      });
      updateUser(user, ["name", "avatar_url"]);
      setName(user.name);
      setAvatar(user.avatar_url);
      setMessage("Profile saved.");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function changeSaved(id: string, remove = false) {
    setMutation(id);
    setError("");
    try {
      if (remove) await unsaveTicket(id);
      else await saveTicket(id, title.trim());
      setEditing("");
      setReload((x) => x + 1);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setMutation("");
    }
  }
  if (!session) return null;
  return (
    <div className={styles.page}>
      <header>
        <p className={styles.eyebrow}>Your workspace</p>
        <h1>Profile</h1>
        <p>Personal details, display preferences and tickets worth keeping.</p>
      </header>
      {(error || preferencesError) && (
        <Notice error>{error || preferencesError}</Notice>
      )}
      {message && <Notice>{message}</Notice>}
      <div className={styles.columns}>
        <section className={styles.panel}>
          <h2>Your profile</h2>
          <div className={styles.identity}>
            <Avatar user={{ name, avatar_url: avatar }} size={64} />
            <div>
              <strong>{session.user.name}</strong>
              <p>
                {session.user.email} · {session.user.role}
              </p>
            </div>
          </div>
          <form onSubmit={submit}>
            <label>
              Name
              <input
                className={fieldClass}
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                maxLength={200}
              />
            </label>
            <label>
              Profile image URL
              <input
                className={fieldClass}
                type="url"
                placeholder="https://…"
                value={avatar}
                onChange={(e) => setAvatar(e.target.value)}
                onPaste={(e) => {
                  const sources = clipboardImageSources(
                    e.clipboardData.getData("text/html"),
                  );
                  if (sources[0]) {
                    e.preventDefault();
                    setAvatar(sources[0]);
                  }
                }}
                maxLength={2048}
              />
            </label>
            <p className={styles.hint}>
              Use a direct HTTPS image URL (“Copy image address”), not a Tenor
              webpage link. Leave empty to use your initials.
            </p>
            <Button primary disabled={busy}>
              {busy ? "Saving…" : "Save profile"}
            </Button>
          </form>
        </section>
        <section className={styles.panel}>
          <h2>Display preferences</h2>
          <fieldset disabled={preferencesBusy}>
            <legend>Theme</legend>
            {(["light", "dark"] as const).map((value) => (
              <label className={styles.option} key={value}>
                <input
                  type="radio"
                  name="theme"
                  checked={(dark ? "dark" : "light") === value}
                  onChange={() => void setTheme(value).catch(() => {})}
                />
                <span>{value === "light" ? "Light" : "Dark"}</span>
              </label>
            ))}
          </fieldset>
          <fieldset disabled={preferencesBusy}>
            <legend>Issues layout</legend>
            {(["modern", "classic"] as const).map((value) => (
              <label className={styles.option} key={value}>
                <input
                  type="radio"
                  name="appearance"
                  checked={appearance === value}
                  onChange={() => void setAppearance(value).catch(() => {})}
                />
                <span>
                  {value === "modern" ? "Modern" : "Classic"}
                  <small>
                    {value === "modern"
                      ? "Clean, spacious ticket table."
                      : "Compact rows inspired by the previous system."}
                  </small>
                </span>
              </label>
            ))}
          </fieldset>
          <p className={styles.hint}>Preferences are saved to your account.</p>
        </section>
      </div>
      <section className={styles.panel}>
        <div className={styles.savedHeading}>
          <div>
            <h2>Saved tickets</h2>
            <p>
              Private to you. Personal titles do not change the original ticket.
            </p>
          </div>
          <input
            className={fieldClass}
            aria-label="Search saved tickets"
            placeholder="Search saved tickets…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        {loading ? (
          <p role="status">Loading saved tickets…</p>
        ) : data?.items.length ? (
          <ul className={styles.saved}>
            {data.items.map((ticket) => (
              <li key={ticket.id}>
                <div className={styles.savedRow}>
                  <div>
                    <Link
                      to="/tickets/$ticketId"
                      params={{ ticketId: ticket.id }}
                    >
                      <strong>
                        #{ticket.number} · {ticket.display_title}
                      </strong>
                    </Link>
                    <p>
                      {ticket.client_name} / {ticket.database_name} ·{" "}
                      {ticket.check_type} · Saved{" "}
                      {new Date(ticket.saved_at).toLocaleDateString()}
                    </p>
                  </div>
                  <StatusBadge status={ticket.status} />
                </div>
                {editing === ticket.id ? (
                  <form
                    className={styles.edit}
                    onSubmit={(e) => {
                      e.preventDefault();
                      void changeSaved(ticket.id);
                    }}
                  >
                    <input
                      className={fieldClass}
                      aria-label={`Personal title for ticket ${ticket.number}`}
                      placeholder={ticket.title}
                      maxLength={200}
                      value={title}
                      onChange={(e) => setTitle(e.target.value)}
                      autoFocus
                    />
                    <Button primary disabled={!!mutation}>
                      Save title
                    </Button>
                    <Button
                      type="button"
                      disabled={!!mutation}
                      onClick={() => setEditing("")}
                    >
                      Cancel
                    </Button>
                  </form>
                ) : (
                  <div className={styles.actions}>
                    <Button
                      disabled={!!mutation}
                      onClick={() => {
                        setEditing(ticket.id);
                        setTitle(ticket.personal_title);
                      }}
                    >
                      Edit personal title
                    </Button>
                    <Button
                      disabled={!!mutation}
                      onClick={() => void changeSaved(ticket.id, true)}
                    >
                      Unsave
                    </Button>
                  </div>
                )}
              </li>
            ))}
          </ul>
        ) : (
          <p>No saved tickets{search ? " match your search" : " yet"}.</p>
        )}
        {data && data.total > 50 && (
          <div className={styles.pagination}>
            <Button
              disabled={page === 1 || loading}
              onClick={() => setPage((x) => x - 1)}
            >
              Previous
            </Button>
            <span>
              Page {page} of {Math.ceil(data.total / 50)}
            </span>
            <Button
              disabled={page * 50 >= data.total || loading}
              onClick={() => setPage((x) => x + 1)}
            >
              Next
            </Button>
          </div>
        )}
      </section>
    </div>
  );
}
