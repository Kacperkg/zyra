import { createFileRoute } from "@tanstack/react-router";
import { DatabasePage } from "../pages/databases/DatabasePage";
export const Route = createFileRoute("/databases/$databaseId")({
  component: () => {
    const { databaseId } = Route.useParams();
    return <DatabasePage key={databaseId} databaseId={databaseId} />;
  },
});
