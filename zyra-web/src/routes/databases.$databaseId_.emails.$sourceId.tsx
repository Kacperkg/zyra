import { createFileRoute } from "@tanstack/react-router";
import { EmailSourcePage } from "../pages/databases/EmailSourcePage";
export const Route = createFileRoute(
  "/databases/$databaseId_/emails/$sourceId",
)({
  component: () => {
    const { databaseId, sourceId } = Route.useParams();
    return (
      <EmailSourcePage
        key={`${databaseId}/${sourceId}`}
        databaseId={databaseId}
        sourceId={sourceId}
      />
    );
  },
});
