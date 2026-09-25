import { createRootRoute } from "@tanstack/react-router";
import { AppLayout } from "../components/layout/AppLayout";
export const Route = createRootRoute({
  component: AppLayout,
  notFoundComponent: () => (
    <p>
      Page not found. <a href="/">Return to dashboard</a>
    </p>
  ),
});
