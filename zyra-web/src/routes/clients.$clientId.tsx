import { createFileRoute } from "@tanstack/react-router";
import { ClientPage } from "../pages/clients/ClientPage";
export const Route = createFileRoute("/clients/$clientId")({
  component: () => {
    const { clientId } = Route.useParams();
    return <ClientPage key={clientId} clientId={clientId} />;
  },
});
