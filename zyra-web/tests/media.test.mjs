import test from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";

const source = stripTypeScriptTypes(
  await readFile(
    new URL("../src/components/editor/clipboard.ts", import.meta.url),
    "utf8",
  ),
);
const { validateImageURL } = await import(
  `data:text/javascript;base64,${Buffer.from(source).toString("base64")}`
);

test("avatar validation rejects unsafe schemes, credentials and Tenor webpage addresses", async () => {
  for (const url of [
    "javascript:alert(1)",
    "http://example.com/image.gif",
    "https://user:password@example.com/image.gif",
    "invalid",
  ])
    await assert.rejects(validateImageURL(url), /HTTPS/);
  await assert.rejects(
    validateImageURL(
      "https://tenor.com/en-GB/view/shrek-donkey-gif-12258072527261586773",
    ),
    /Tenor webpage/,
  );
  await validateImageURL("");
});

test("avatar validation tests image loading and cleans up handlers", async () => {
  globalThis.window = { setTimeout };
  let image;
  globalThis.Image = class {
    constructor() {
      image = this;
    }
    set src(value) {
      this.url = value;
      queueMicrotask(() => this.onload());
    }
    removeAttribute(name) {
      assert.equal(name, "src");
    }
  };
  try {
    await validateImageURL("https://example.com/avatar.gif");
    assert.equal(image.referrerPolicy, "no-referrer");
    assert.equal(image.onload, null);
    assert.equal(image.onerror, null);
  } finally {
    delete globalThis.window;
    delete globalThis.Image;
  }
});

test("avatar validation explains image-load failures rather than silently saving", async () => {
  globalThis.window = { setTimeout };
  globalThis.Image = class {
    set src(value) {
      queueMicrotask(() => this.onerror());
    }
    removeAttribute() {}
  };
  try {
    await assert.rejects(
      validateImageURL("https://example.com/page"),
      /could not load as an image/,
    );
  } finally {
    delete globalThis.window;
    delete globalThis.Image;
  }
});
