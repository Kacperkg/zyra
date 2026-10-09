import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
const source = stripTypeScriptTypes(
  await readFile(
    new URL("../src/components/editor/content.ts", import.meta.url),
    "utf8",
  ),
);
const {
  safeURL,
  autolink,
  groupBlocks,
  hasContent,
  emptyContent,
  directGIFURL,
  tenorPageURL,
} = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
);
test("Tenor page detection is exact and keeps the large GIF ID as text", () => {
  const url =
    "https://tenor.com/en-GB/view/burro-do-shrek-gif-18347208884916589175";
  assert.equal(tenorPageURL(url), url);
  assert.equal(
    tenorPageURL("https://www.tenor.com/view/example-123?share=1"),
    "https://www.tenor.com/view/example-123?share=1",
  );
  for (const value of [
    "https://tenor.com.evil.test/view/a-123",
    "https://tenor.com@127.0.0.1/view/a-123",
    "http://tenor.com/view/a-123",
    "https://tenor.com:8080/view/a-123",
    "https://tenor.com/view/a-x",
    "https://tenor.com/search/cats",
  ])
    assert.equal(tenorPageURL(value), null);
});
test("direct HTTPS GIF URLs become media while page links and explicitly linked text stay links", () => {
  const url = "https://media1.tenor.com/m/_p5hRe1YEncAAAAC/burro-do-shrek.gif";
  assert.equal(directGIFURL(url), url);
  assert.equal(
    directGIFURL("https://example.com/reaction.GIF?size=small"),
    "https://example.com/reaction.GIF?size=small",
  );
  for (const value of [
    "http://example.com/a.gif",
    "https://tenor.com/view/example",
    "https://example.com/page?image=a.gif",
    "https://user:pass@example.com/a.gif",
  ])
    assert.equal(directGIFURL(value), null);
  assert.deepEqual(autolink({ type: "text", text: `Before ${url}. After` }), [
    { type: "text", text: "Before " },
    { type: "gif", src: url },
    { type: "text", text: ". After" },
  ]);
  const explicit = { type: "text", text: url, href: url };
  assert.deepEqual(autolink(explicit), [explicit]);
});
test("URL validation excludes script URLs, credentials and non-HTTPS GIF URLs", () => {
  for (const value of [
    "javascript:alert(1)",
    "data:image/gif;base64,AA",
    "https://user:password@example.com/a.gif",
    "//example.com/a.gif",
  ])
    assert.equal(safeURL(value), null);
  assert.equal(safeURL("http://example.com/a.gif", true), null);
  assert.equal(
    safeURL("https://example.com/media", true),
    "https://example.com/media",
  );
});
test("automatic links preserve marks and sentence punctuation", () => {
  const result = autolink({
    type: "text",
    text: "Read https://example.com/runbook. Then http://example.org/help!",
    bold: true,
  });
  assert.equal(
    result.map((node) => node.text).join(""),
    "Read https://example.com/runbook. Then http://example.org/help!",
  );
  assert.deepEqual(
    result.filter((node) => node.href).map((node) => node.href),
    ["https://example.com/runbook", "http://example.org/help"],
  );
  assert.ok(result.every((node) => node.bold));
});
test("grouped ordered items retain a continuous numbered list and paragraph boundaries", () => {
  const block = (type) => ({
    type,
    children: [{ type: "text", text: "Item" }],
  });
  const groups = groupBlocks([
    block("ordered_list"),
    block("ordered_list"),
    block("paragraph"),
    block("ordered_list"),
    block("bullet_list"),
  ]);
  assert.deepEqual(
    groups.map((group) => group.length),
    [2, 1, 1, 1],
  );
});
test("GIF-only and mention-only comments qualify while empty comments do not", () => {
  assert.equal(hasContent(emptyContent()), false);
  assert.equal(
    hasContent({
      blocks: [
        { type: "paragraph", children: [{ type: "text", text: " \n " }] },
      ],
    }),
    false,
  );
  for (const node of [
    { type: "gif", src: "https://example.com/reaction" },
    { type: "mention", user_id: "person" },
  ])
    assert.equal(
      hasContent({ blocks: [{ type: "paragraph", children: [node] }] }),
      true,
    );
});
