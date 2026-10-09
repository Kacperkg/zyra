import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
const source = stripTypeScriptTypes(
  await readFile(
    new URL("../src/types/ticket-table.ts", import.meta.url),
    "utf8",
  ),
);
const { ticketColumns, ticketTimestamp, assessmentLabel } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
);
test("Classic matches the seven reference columns while Modern retains ticket status", () => {
  assert.deepEqual(
    ticketColumns("classic").map((col) => col.label),
    [
      "#",
      "Check type",
      "Created Date",
      "Created Time",
      "Client",
      "Database",
      "Assessment Type",
    ],
  );
  assert.equal(ticketColumns("modern").at(-1).id, "status");
});
test("Only date sorts the timestamp; all exposed sort keys are supported by the API", () => {
  const supported = new Set([
    "number",
    "check_type",
    "created_at",
    "client",
    "database",
    "assessment_type",
    "status",
  ]);
  for (const appearance of ["modern", "classic"]) {
    const cols = ticketColumns(appearance);
    assert.equal(cols.find((col) => col.id === "created_time").sort, undefined);
    const keys = cols.flatMap((col) => (col.sort ? [col.sort] : []));
    assert.equal(new Set(keys).size, keys.length);
    assert.ok(keys.every((key) => supported.has(key)));
  }
});
test("Classic dates use fixed year-month-day and a full 24-hour clock", () => {
  const local = new Date(2026, 8, 24, 21, 5, 26);
  const result = ticketTimestamp(local.toISOString(), "classic");
  assert.equal(result.date, "2026-09-24");
  assert.equal(result.time, "21:05:26");
  assert.equal(
    ticketTimestamp(new Date(2026, 8, 24, 0, 0, 0).toISOString(), "classic")
      .time,
    "00:00:00",
  );
  assert.deepEqual(ticketTimestamp("invalid", "classic"), {
    date: "Unavailable",
    time: "Unavailable",
  });
});
test("Assessment labels remain readable including an unknown future check", () => {
  assert.equal(assessmentLabel("daily_check"), "Daily check");
  assert.equal(assessmentLabel("missing"), "Missing email check");
  assert.equal(assessmentLabel("future_check"), "future check");
});
