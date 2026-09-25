import { useState, type FormEvent } from "react";
import { Navigate, useNavigate } from "@tanstack/react-router";
import { useAuth } from "../../contexts/AuthContext";
import { useTheme } from "../../contexts/ThemeContext";
import { Button, Notice, fieldClass } from "../../components/ui/Controls";
import styles from "./LoginPage.module.css";
export function LoginPage() {
  const { session, login } = useAuth();
  const { dark, toggle } = useTheme();
  const navigate = useNavigate();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  if (session) return <Navigate to="/" />;
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await login(identifier, password);
      await navigate({ to: "/" });
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <main className={styles.page}>
      <div className={styles.theme}>
        <Button onClick={toggle}>{dark ? "Light" : "Dark"} theme</Button>
      </div>
      <section className={styles.card}>
        <div className={styles.wordmark}>
          Zyra<span>Database health, in focus.</span>
        </div>
        <h1>Welcome back</h1>
        <p>Sign in to review your database checks.</p>
        <form onSubmit={submit}>
          <label>
            Username or email
            <input
              className={fieldClass}
              autoComplete="username"
              value={identifier}
              onChange={(e) => setIdentifier(e.target.value)}
              required
            />
          </label>
          <label>
            Password
            <input
              className={fieldClass}
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </label>
          {error && <Notice error>{error}</Notice>}
          <Button primary disabled={busy}>
            {busy ? "Signing in…" : "Sign in"}
          </Button>
        </form>
        <p className={styles.help}>
          Need account access or a password reset? Contact your administrator.
        </p>
      </section>
    </main>
  );
}
