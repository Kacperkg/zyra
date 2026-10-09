import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import {
  login,
  logout,
  watchSession,
  updateSessionUser,
  getSession,
  setSession,
} from "../api/client";
import type { Session } from "../types/api";
const AuthContext = createContext<{
  session: Session | null;
  login: typeof login;
  logout: typeof logout;
  updateUser: typeof updateSessionUser;
} | null>(null);
export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, updateSession] = useState<Session | null>(getSession);
  useEffect(() => watchSession(updateSession), []);
  useEffect(() => {
    if (!session) return;
    const timer = window.setTimeout(
      () => setSession(null),
      Math.max(0, Date.parse(session.session_expires_at) - Date.now()),
    );
    return () => clearTimeout(timer);
  }, [session?.session_expires_at]);
  return (
    <AuthContext.Provider
      value={{ session, login, logout, updateUser: updateSessionUser }}
    >
      {children}
    </AuthContext.Provider>
  );
}
export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("AuthProvider missing");
  return value;
}
