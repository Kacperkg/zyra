export function ActionIcon({ name }: { name: "edit" | "delete" | "reopen" | "close" }) {
  const paths = {
    edit: "m15 4 5 5M4 20l5-1L20 8a2 2 0 0 0-5-5L4 14v6Z",
    delete: "M3 6h18M9 6V3h6v3M5 6l1 15h12l1-15M10 10v7M14 10v7",
    reopen: "M3 10a9 9 0 1 1 1 8M3 4v6h6",
    close: "m5 12 4 4L19 6",
  };
  return (
    <svg
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={paths[name]} />
    </svg>
  );
}
