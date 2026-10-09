import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
const source = stripTypeScriptTypes(
  await readFile(
    new URL("../src/types/ticket-query.ts", import.meta.url),
    "utf8",
  ),
);
const { buildTicketQuery } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
);
test("Each 100-ticket screen uses two 50-summary requests, then advances the API page", () => {
  for (const [screen, first, second] of [
    [1, 1, 2],
    [2, 3, 4],
    [3, 5, 6],
  ]) {
    for (const [batch, expected] of [
      [1, first],
      [2, second],
    ]) {
      const params = buildTicketQuery("open", { page: screen }, batch);
      assert.equal(params.get("limit"), "50");
      assert.equal(params.get("page"), String(expected));
      assert.equal(params.get("status"), "open");
    }
  }
});
test("Both scroll batches preserve all server filter and sort parameters", () => {
  const search = {
    page: 2,
    q: "Acme / FRA",
    check: "fra",
    client_id: "client-1",
    database_id: "db-1",
    assessment_type: "daily_check",
    number: "52",
    created_from: "2026-09-20",
    created_to: "2026-09-24",
    sort: "database",
    order: "asc",
  };
  for (const batch of [1, 2]) {
    const params = buildTicketQuery("closed", search, batch);
    assert.equal(params.get("status"), "closed");
    assert.equal(params.get("check_type"), "fra");
    for (const key of [
      "q",
      "client_id",
      "database_id",
      "assessment_type",
      "number",
      "created_from",
      "created_to",
      "sort",
      "order",
    ])
      assert.equal(params.get(key), search[key]);
    assert.equal(params.has("check"), false);
    assert.equal(params.has("appearance"), false);
  }
});
test("Fresh requests default to newest open tickets and omit empty filters", () => {
  const params = buildTicketQuery(
    undefined,
    { q: "", check: "", client_id: "", order: "invalid" },
    1,
  );
  assert.deepEqual(Object.fromEntries(params), {
    status: "open",
    limit: "50",
    page: "1",
    sort: "created_at",
    order: "desc",
  });
});
