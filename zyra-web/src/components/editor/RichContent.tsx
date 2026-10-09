import { useState, type CSSProperties } from "react";
import {
  autolink,
  groupBlocks,
  safeURL,
  type RichContent as Document,
  type RichNode,
} from "./content";
import styles from "./RichEditor.module.css";
function Gif({ node }: { node: RichNode }) {
  const [failed, setFailed] = useState(false);
  const src = safeURL(node.src || "", true);
  return failed || !src ? (
    <span className={styles.unavailable}>
      GIF unavailable{node.alt ? `: ${node.alt}` : ""}
    </span>
  ) : (
    <img
      className={styles.gif}
      src={src}
      alt={node.alt || "GIF"}
      loading="lazy"
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
    />
  );
}
export function RichContent({
  content,
  names = {},
}: {
  content: Document;
  names?: Record<string, string>;
}) {
  function renderNode(node: RichNode, key: string) {
    if (node.type === "gif") return <Gif key={key} node={node} />;
    if (node.type === "mention")
      return (
        <span key={key} className={styles.mention}>
          @{names[node.user_id || ""] || "Team member"}
        </span>
      );
    const style: CSSProperties = {
      fontWeight: node.bold ? 700 : undefined,
      fontStyle: node.italic ? "italic" : undefined,
      textDecoration: [
        node.underline && "underline",
        node.strike && "line-through",
      ]
        .filter(Boolean)
        .join(" "),
      fontSize: node.size,
      color: /^#[\da-f]{6}$/i.test(node.color || "") ? node.color : undefined,
    };
    const href = safeURL(node.href || "");
    return href ? (
      <a
        key={key}
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        style={style}
      >
        {node.text}
      </a>
    ) : (
      <span key={key} style={style}>
        {node.text}
      </span>
    );
  }
  return (
    <div className={styles.rendered}>
      {groupBlocks(content.blocks).map((group, index) => {
        const block = group[0];
        const children = block.children
          .flatMap(autolink)
          .map((node, i) => renderNode(node, `${index}-${i}`));
        const items = group.map((item, j) => (
          <li key={j}>
            {item.children
              .flatMap(autolink)
              .map((node, i) => renderNode(node, `${index}-${j}-${i}`))}
          </li>
        ));
        const props = { key: index, style: { textAlign: block.align } };
        return block.type === "code_block" ? (
          <pre {...props}>{children}</pre>
        ) : block.type === "bullet_list" ? (
          <ul {...props}>{items}</ul>
        ) : block.type === "ordered_list" ? (
          <ol {...props}>{items}</ol>
        ) : (
          <p {...props}>{children}</p>
        );
      })}
    </div>
  );
}
