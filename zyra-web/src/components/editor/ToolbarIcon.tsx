// Compact Office-style formatting symbols; buttons provide accessible names.
export function ToolbarIcon({ name }: { name: string }) {
  const paths: Record<string, string> = {
    bold: "M7 4h5a4 4 0 0 1 0 8H7m0 0h6a4 4 0 0 1 0 8H7V4",
    italic: "M10 4h8M6 20h8M15 4 9 20",
    underline: "M6 4v7a6 6 0 0 0 12 0V4M4 21h16",
    strikeThrough: "M17 5c-2-2-10-2-10 3 0 2 2 3 5 4m-8 0h16m-3 3c0 6-9 6-11 2",
    insertUnorderedList: "M9 6h12M9 12h12M9 18h12M3 6h1M3 12h1M3 18h1",
    insertOrderedList: "M10 6h11M10 12h11M10 18h11M3 4h2v5M3 9h4M3 13h4l-4 6h4",
    justifyLeft: "M3 5h18M3 10h12M3 15h18M3 20h12",
    justifyCenter: "M3 5h18M6 10h12M3 15h18M6 20h12",
    justifyRight: "M3 5h18M9 10h12M3 15h18M9 20h12",
    code: "m8 6-6 6 6 6m8-12 6 6-6 6m-3-15-2 18",
    link: "m10 14 4-4m-5 7-2 2a4 4 0 0 1-6-6l5-5a4 4 0 0 1 6 0m3-1 2-2a4 4 0 0 1 6 6l-5 5a4 4 0 0 1-6 0",
    gif: "M4 3h16v18H4V3m1 13 5-5 4 4 3-3 3 3M8 7h.01",
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
