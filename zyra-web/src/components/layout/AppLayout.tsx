import { Outlet, Navigate, useLocation } from "@tanstack/react-router";
import { useAuth } from "../../contexts/AuthContext";
import { useAppearance } from "../../contexts/ThemeContext";
import { MainNavigation } from "../navigation/MainNavigation";
import styles from "./AppLayout.module.css";
export function AppLayout() {
  const { session } = useAuth();
  const { appearance } = useAppearance();
  const location = useLocation();
  const wideClassicList =
    appearance === "classic" &&
    /^\/issues\/(open|closed)\/?$/.test(location.pathname);
  if (location.pathname === "/login") return <Outlet />;
  if (!session) return <Navigate to="/login" />;
  return (
    <>
      <MainNavigation />
      <main className={`${styles.main} ${wideClassicList ? styles.wide : ""}`}>
        <Outlet />
      </main>
    </>
  );
}
