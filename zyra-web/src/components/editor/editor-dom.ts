import { groupBlocks, readEditor, safeURL, type RichContent } from "./content";
import styles from "./RichEditor.module.css";

export function createGif(src: string, alt = "") {
  const image = document.createElement("img");
  image.src = src;
  image.alt = alt;
  image.className = styles.gif;
  image.referrerPolicy = "no-referrer";
  return image;
}

export function hasGifCapacity(root: HTMLElement | null, added: number) {
  const count = root
    ? readEditor(root)
        .blocks.flatMap((b) => b.children)
        .filter((n) => n.type === "gif").length
    : 0;
  return count + added <= 10;
}

export function populateEditor(
  root: HTMLElement,
  content: RichContent,
  names: Record<string, string>,
) {
  root.replaceChildren();
  for (const group of groupBlocks(content.blocks)) {
    const list =
      group[0].type === "bullet_list" || group[0].type === "ordered_list"
        ? document.createElement(group[0].type === "bullet_list" ? "ul" : "ol")
        : null;
    if (list) root.append(list);
    for (const block of group) {
      const element = document.createElement(
        block.type === "code_block"
          ? "pre"
          : block.type === "paragraph"
            ? "p"
            : "li",
      );
      element.style.textAlign = block.align || "left";
      for (const node of block.children) {
        if (node.type === "mention") {
          const span = document.createElement("span");
          span.dataset.mention = node.user_id;
          span.contentEditable = "false";
          span.className = styles.mention;
          span.textContent = "@" + (names[node.user_id || ""] || "Team member");
          element.append(span);
        } else if (node.type === "gif") {
          const src = safeURL(node.src || "", true);
          if (!src) continue;
          element.append(createGif(src, node.alt));
        } else {
          const href = safeURL(node.href || "");
          const span = document.createElement(href ? "a" : "span");
          span.textContent = node.text || "";
          if (href) span.setAttribute("href", href);
          if (node.bold) {
            const strong = document.createElement("b");
            strong.append(span);
            element.append(strong);
          } else element.append(span);
          if (node.italic) span.style.fontStyle = "italic";
          if (node.underline || node.strike)
            span.style.textDecoration = [
              node.underline && "underline",
              node.strike && "line-through",
            ]
              .filter(Boolean)
              .join(" ");
          if (node.size) span.style.fontSize = `${node.size}px`;
          if (/^#[\da-f]{6}$/i.test(node.color || ""))
            span.style.color = node.color!;
        }
      }
      if (list) list.append(element);
      else root.append(element);
    }
  }
}
