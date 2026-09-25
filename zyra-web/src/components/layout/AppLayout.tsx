import { Outlet, Navigate, useLocation } from "@tanstack/react-router";
import { useAuth } from "../../contexts/AuthContext";
import { MainNavigation } from "../navigation/MainNavigation";
import styles from "./AppLayout.module.css";
export function AppLayout() {
  const { session } = useAuth();
  const location = useLocation();
  if (location.pathname === "/login") return <Outlet />;
  if (!session) return <Navigate to="/login" />;
  return (
    <>
      <MainNavigation />
      <main className={styles.main}>
        <Outlet />
      </main>
    </>
  );
}
