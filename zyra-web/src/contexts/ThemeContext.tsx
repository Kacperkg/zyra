import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useAuth } from "./AuthContext";
import { usersApi, type ProfileInput } from "../api/users.api";
type Preferences = {
  dark: boolean;
  appearance: "modern" | "classic";
  busy: boolean;
  error: string;
  toggle: () => void;
  setTheme: (theme: "light" | "dark") => Promise<void>;
  setAppearance: (appearance: "modern" | "classic") => Promise<void>;
};
const ThemeContext = createContext<Preferences | null>(null);
export function ThemeProvider({ children }: { children: ReactNode }) {
  const { session, updateUser } = useAuth();
  const [dark, setDark] = useState(false);
  const [appearance, setLocalAppearance] = useState<"modern" | "classic">(
    "modern",
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const generation = useRef(0);
  const queue = useRef(Promise.resolve());
  useEffect(() => {
    generation.current++;
    setError("");
    setBusy(false);
  }, [session?.user.id]);
  useEffect(() => {
    if (session) {
      setDark(session.user.theme === "dark");
      setLocalAppearance(
        session.user.appearance === "classic" ? "classic" : "modern",
      );
    }
  }, [session?.user.theme, session?.user.appearance, session?.user.id]);
  useEffect(() => {
    document.documentElement.dataset.theme = dark ? "dark" : "light";
    document.documentElement.dataset.appearance = appearance;
  }, [dark, appearance]);
  async function persist(input: ProfileInput) {
    if (!session) {
      if (input.theme) setDark(input.theme === "dark");
      if (input.appearance) setLocalAppearance(input.appearance);
      return;
    }
    const expected = generation.current;
    setBusy(true);
    setError("");
    const operation = queue.current
      .catch(() => {})
      .then(async () => {
        if (expected !== generation.current) return;
        const user = await usersApi.update(input);
        if (expected === generation.current)
          updateUser(user, Object.keys(input) as (keyof typeof user)[]);
      });
    queue.current = operation;
    try {
      await operation;
    } catch (e) {
      if (expected === generation.current) setError((e as Error).message);
      throw e;
    } finally {
      if (expected === generation.current) setBusy(false);
    }
  }
  return (
    <ThemeContext.Provider
      value={{
        dark,
        appearance,
        busy,
        error,
        toggle: () => {
          void persist({ theme: dark ? "light" : "dark" }).catch(() => {});
        },
        setTheme: (theme) => persist({ theme }),
        setAppearance: (value) => persist({ appearance: value }),
      }}
    >
      {children}
    </ThemeContext.Provider>
  );
}
export function useTheme() {
  const value = useContext(ThemeContext);
  if (!value) throw new Error("ThemeProvider missing");
  return value;
}
export const useAppearance = useTheme;
