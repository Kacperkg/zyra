import type {
  CommentInline,
  CommentBlock,
  CommentContent,
} from "../../types/api";
export type RichNode = CommentInline;
export type RichBlock = CommentBlock;
export type RichContent = CommentContent;
export const emptyContent = (): RichContent => ({
  blocks: [{ type: "paragraph", children: [] }],
});
export const hasContent = (content: RichContent) =>
  content.blocks.some((block) =>
    block.children.some((node) => node.type !== "text" || !!node.text?.trim()),
  );
export function groupBlocks(blocks: RichBlock[]): RichBlock[][] {
  const groups: RichBlock[][] = [];
  for (const block of blocks) {
    const last = groups[groups.length - 1];
    if (
      last &&
      (block.type === "bullet_list" || block.type === "ordered_list") &&
      last[0].type === block.type &&
      last[0].align === block.align
    )
      last.push(block);
    else groups.push([block]);
  }
  return groups;
}
export function safeURL(value: string, httpsOnly = false) {
  try {
    const url = new URL(value);
    return !url.username &&
      !url.password &&
      (url.protocol === "https:" || (!httpsOnly && url.protocol === "http:")) &&
      value.length <= 2048
      ? url.href
      : null;
  } catch {
    return null;
  }
}
export function autolink(node: RichNode): RichNode[] {
  if (node.type !== "text" || node.href) return [node];
  const text = node.text || "";
  const nodes: RichNode[] = [];
  let cursor = 0;
  for (const match of text.matchAll(/https?:\/\/[^\s<>]+/gi)) {
    const start = match.index!;
    const value = match[0].replace(/[.,;!?\)\]]+$/, "");
    const href = safeURL(value);
    if (!href) continue;
    if (start > cursor)
      nodes.push({ ...node, text: text.slice(cursor, start) });
    const gif = directGIFURL(value);
    nodes.push(
      gif ? { type: "gif", src: gif } : { ...node, text: value, href },
    );
    cursor = start + value.length;
  }
  if (cursor < text.length) nodes.push({ ...node, text: text.slice(cursor) });
  return nodes.length ? nodes : [node];
}
export function directGIFURL(value: string): string | null {
  const url = safeURL(value, true);
  return url && /\.gif$/i.test(new URL(url).pathname) ? url : null;
}
export function tenorPageURL(value: string): string | null {
  const url = safeURL(value.trim(), true);
  if (!url) return null;
  const parsed = new URL(url);
  return ["tenor.com", "www.tenor.com"].includes(parsed.hostname) &&
    !parsed.port &&
    /^\/(?:[a-z]{2}(?:-[a-z]{2})?\/)?view\/[a-z0-9_-]+-\d{1,20}\/?$/i.test(
      parsed.pathname,
    )
    ? url
    : null;
}
export function readEditor(root: HTMLElement): RichContent {
  const blocks: RichBlock[] = [];
  function inline(node: Node, marks: Partial<RichNode> = {}): RichNode[] {
    if (node.nodeType === Node.TEXT_NODE)
      return autolink({ type: "text", text: node.textContent || "", ...marks });
    if (!(node instanceof HTMLElement)) return [];
    if (node.dataset.mention)
      return [{ type: "mention", user_id: node.dataset.mention }];
    if (node instanceof HTMLImageElement) {
      const src = safeURL(node.src, true);
      return src ? [{ type: "gif", src, alt: node.alt }] : [];
    }
    if (node.tagName === "BR") return [{ type: "text", text: "\n", ...marks }];
    const next = { ...marks };
    if (
      node.style.fontWeight === "bold" ||
      Number(node.style.fontWeight) >= 600
    )
      next.bold = true;
    if (node.style.fontStyle === "italic") next.italic = true;
    if (node.style.textDecoration.includes("underline")) next.underline = true;
    if (node.style.textDecoration.includes("line-through")) next.strike = true;
    if (["B", "STRONG"].includes(node.tagName)) next.bold = true;
    if (["I", "EM"].includes(node.tagName)) next.italic = true;
    if (node.tagName === "U") next.underline = true;
    if (["S", "STRIKE", "DEL"].includes(node.tagName)) next.strike = true;
    if (node.tagName === "A") {
      const href = safeURL(node.getAttribute("href") || "");
      if (href) next.href = href;
    }
    const color = node.style.color || node.getAttribute("color");
    if (color) {
      const rgb = color.match(/^rgb\((\d+),\s*(\d+),\s*(\d+)\)$/);
      const hex = rgb
        ? "#" +
          rgb
            .slice(1)
            .map((v) => Number(v).toString(16).padStart(2, "0"))
            .join("")
        : color;
      if (/^#[\da-f]{6}$/i.test(hex)) next.color = hex;
    }
    const fontSizes: Record<string, number> = {
      "1": 12,
      "2": 14,
      "3": 16,
      "4": 18,
      "5": 24,
    };
    const size =
      parseInt(node.style.fontSize) ||
      fontSizes[node.getAttribute("size") || ""];
    if ([12, 14, 16, 18, 20, 24].includes(size))
      next.size = size as RichNode["size"];
    return [...node.childNodes].flatMap((child) => inline(child, next));
  }
  function visit(node: Node) {
    if (node instanceof HTMLElement && ["UL", "OL"].includes(node.tagName)) {
      [...node.children].forEach(visit);
      return;
    }
    if (
      node instanceof HTMLElement &&
      ["P", "DIV", "LI", "PRE"].includes(node.tagName)
    ) {
      if (
        node.children.length &&
        [...node.children].some((child) =>
          ["P", "DIV", "UL", "OL"].includes(child.tagName),
        )
      ) {
        [...node.childNodes].forEach(visit);
        return;
      }
      const align = node.style.textAlign;
      blocks.push({
        type:
          node.tagName === "PRE"
            ? "code_block"
            : node.tagName === "LI"
              ? node.parentElement?.tagName === "OL"
                ? "ordered_list"
                : "bullet_list"
              : "paragraph",
        ...(align === "left" || align === "center" || align === "right"
          ? { align }
          : {}),
        children: [...node.childNodes].flatMap((child) => inline(child)),
      });
    } else {
      if (!blocks.length || blocks[blocks.length - 1].type !== "paragraph")
        blocks.push({ type: "paragraph", children: [] });
      blocks[blocks.length - 1].children.push(...inline(node));
    }
  }
  [...root.childNodes].forEach(visit);
  return { blocks: blocks.length ? blocks : emptyContent().blocks };
}
