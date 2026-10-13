import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
const source = stripTypeScriptTypes(await readFile(new URL("../src/types/configuration-validation.ts", import.meta.url), "utf8"));
const { optionalNumber, resourceRule, receivedTimestamp } = await import(`data:text/javascript;base64,${Buffer.from(source).toString("base64")}`);
test("optional percentage thresholds preserve unset and zero distinctly", () => {
  assert.equal(optionalNumber("", "percentage"), null);
  assert.equal(optionalNumber("0", "percentage"), 0);
  assert.equal(optionalNumber("90", "percentage"), 90);
  assert.equal(optionalNumber("0.5", "percentage"), .5);
  for (const invalid of ["-1", "101", "Infinity", "NaN"]) assert.throws(() => optionalNumber(invalid, "percentage"));
});
test("sequence lag is a bounded whole sequence value, not a percentage", () => {
  assert.equal(optionalNumber("500", "sequences"), 500);
  assert.equal(optionalNumber("0", "sequences"), 0);
  assert.equal(optionalNumber(" ", "sequences"), null);
  for (const invalid of ["-.1", "1.5", "2147483648", "Infinity"]) assert.throws(() => optionalNumber(invalid, "sequences"));
});
test("resource rules have explicit threshold units; backups have only ignore", () => {
  assert.deepEqual(resourceRule(false, "90", "max_used_percent"), { ignore: false, max_used_percent: 90 });
  assert.deepEqual(resourceRule(true, "0", "min_free_percent"), { ignore: true, min_free_percent: 0 });
  assert.deepEqual(resourceRule(false, "", "min_free_percent"), { ignore: false });
  assert.deepEqual(resourceRule(true, "", null), { ignore: true });
});
test("display uses received ISO timestamps without guessing Oracle ReportOn timezone", () => {
  assert.notEqual(receivedTimestamp("2026-10-13T18:00:01Z"), "Unavailable");
  assert.equal(receivedTimestamp("13-oct-2026:18:00:01"), "Unavailable");
});
