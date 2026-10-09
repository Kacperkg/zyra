import { useEffect, useId, useRef, useState } from "react";
import { mentionOptions, type MentionOption } from "../../api/mentions.api";
import { Button } from "../ui/Controls";
import {
  emptyContent,
  groupBlocks,
  readEditor,
  safeURL,
  autolink,
  tenorPageURL,
  type RichContent,
} from "./content";
import styles from "./RichEditor.module.css";
import { ToolbarIcon } from "./ToolbarIcon";
import { clipboardImageSources } from "./clipboard";

export function RichEditor({
  initialContent,
  disabled = false,
  onChange,
  names = {},
  label = "Comment",
}: {
  initialContent?: RichContent;
  disabled?: boolean;
  onChange: (content: RichContent) => void;
  names?: Record<string, string>;
  label?: string;
}) {
  const root = useRef<HTMLDivElement>(null);
  const suggestionId = useId();
  const selection = useRef<Range | null>(null);
  const mentionRange = useRef<Range | null>(null);
  const [tool, setTool] = useState<"link" | "gif" | null>(null);
  const [url, setURL] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [options, setOptions] = useState<MentionOption[]>([]);
  const [selected, setSelected] = useState(0);
  const [searching, setSearching] = useState(false);
  const initial = useRef(initialContent || emptyContent());
  useEffect(() => {
    if (!root.current) return;
    root.current.replaceChildren();
    for (const group of groupBlocks(initial.current.blocks)) {
      const list =
        group[0].type === "bullet_list" || group[0].type === "ordered_list"
          ? document.createElement(
              group[0].type === "bullet_list" ? "ul" : "ol",
            )
          : null;
      if (list) root.current.append(list);
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
            span.textContent =
              "@" + (names[node.user_id || ""] || "Team member");
            element.append(span);
          } else if (node.type === "gif") {
            const src = safeURL(node.src || "", true);
            if (!src) continue;
            const image = document.createElement("img");
            image.src = src;
            image.alt = node.alt || "";
            image.className = styles.gif;
            image.referrerPolicy = "no-referrer";
            element.append(image);
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
        else root.current.append(element);
      }
    }
  }, []);
  useEffect(() => {
    const controller = new AbortController();
    setOptions([]);
    setSelected(0);
    if (query.length < 2) {
      setSearching(false);
      return;
    }
    setSearching(true);
    const timer = window.setTimeout(() => {
      mentionOptions(query, controller.signal)
        .then((items) => {
          if (!controller.signal.aborted) setOptions(items);
        })
        .catch((e) => {
          if (!controller.signal.aborted) setError(e.message);
        })
        .finally(() => {
          if (!controller.signal.aborted) setSearching(false);
        });
    }, 180);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [query]);
  function remember() {
    const value = window.getSelection();
    if (value?.rangeCount && root.current?.contains(value.anchorNode))
      selection.current = value.getRangeAt(0).cloneRange();
  }
  function restore() {
    root.current?.focus();
    const value = window.getSelection();
    if (
      selection.current &&
      root.current?.contains(selection.current.startContainer)
    ) {
      value?.removeAllRanges();
      value?.addRange(selection.current);
    }
  }
  function changed() {
    if (root.current) onChange(readEditor(root.current));
    remember();
    const caret = window.getSelection();
    if (caret?.anchorNode?.nodeType === Node.TEXT_NODE && caret.isCollapsed) {
      const text =
        caret.anchorNode.textContent?.slice(0, caret.anchorOffset) || "";
      const match = text.match(/(?:^|\s)@([\p{L}\p{N} .'-]{0,60})$/u);
      if (match) {
        const range = document.createRange();
        range.setStart(
          caret.anchorNode,
          caret.anchorOffset - match[1].length - 1,
        );
        range.setEnd(caret.anchorNode, caret.anchorOffset);
        mentionRange.current = range;
        setQuery(match[1]);
        return;
      }
    }
    setQuery("");
    mentionRange.current = null;
  }
  function command(name: string, value?: string) {
    restore();
    document.execCommand(name, false, value);
    changed();
  }
  function insert(node: Node) {
    restore();
    // Inserting a fragment moves its children into the document; the fragment
    // itself has no parent and cannot be used as a caret anchor afterwards.
    const lastInserted =
      node instanceof DocumentFragment ? node.lastChild : node;
    const range = window.getSelection()?.rangeCount
      ? window.getSelection()!.getRangeAt(0)
      : null;
    if (range && root.current?.contains(range.commonAncestorContainer)) {
      range.deleteContents();
      range.insertNode(node);
      if (lastInserted) range.setStartAfter(lastInserted);
      range.collapse(true);
      window.getSelection()?.removeAllRanges();
      window.getSelection()?.addRange(range);
    } else root.current?.append(node);
    changed();
  }
  function mention(person: MentionOption) {
    if (!mentionRange.current) return;
    selection.current = mentionRange.current;
    const fragment = document.createDocumentFragment();
    const span = document.createElement("span");
    span.dataset.mention = person.id;
    span.contentEditable = "false";
    span.className = styles.mention;
    span.textContent = `@${person.name}`;
    fragment.append(span, document.createTextNode(" "));
    restore();
    const range = selection.current;
    range.deleteContents();
    range.insertNode(fragment);
    range.setStartAfter(span.nextSibling!);
    range.collapse(true);
    window.getSelection()?.removeAllRanges();
    window.getSelection()?.addRange(range);
    setQuery("");
    mentionRange.current = null;
    changed();
  }
  function addURL() {
    if (tool === "gif" && tenorPageURL(url)) {
      setError(
        "This is a Tenor webpage. Use “Copy image address” for the direct GIF URL instead.",
      );
      return;
    }
    const href = safeURL(url, tool === "gif");
    if (!href) {
      setError(
        tool === "gif"
          ? "Enter a direct HTTPS GIF URL without credentials."
          : "Enter an absolute HTTP or HTTPS link without credentials.",
      );
      return;
    }
    if (tool === "gif") {
      if (
        root.current &&
        readEditor(root.current)
          .blocks.flatMap((b) => b.children)
          .filter((n) => n.type === "gif").length >= 10
      ) {
        setError("A comment can contain up to 10 GIFs.");
        return;
      }
      const image = document.createElement("img");
      image.src = href;
      image.alt = description;
      image.className = styles.gif;
      image.referrerPolicy = "no-referrer";
      insert(image);
    } else {
      const anchor = document.createElement("a");
      anchor.href = href;
      anchor.textContent = description || selection.current?.toString() || url;
      insert(anchor);
    }
    setTool(null);
    setURL("");
    setDescription("");
    setError("");
  }
  return (
    <div className={styles.editor}>
      <div
        className={styles.toolbar}
        role="toolbar"
        aria-label="Comment formatting"
      >
        {[
          ["bold", "Bold", "B"],
          ["italic", "Italic", "I"],
          ["underline", "Underline", "U"],
          ["strikeThrough", "Strikethrough", "S"],
          ["insertUnorderedList", "Bullet list", "• List"],
          ["insertOrderedList", "Numbered list", "1. List"],
          ["justifyLeft", "Align left", "Left"],
          ["justifyCenter", "Align center", "Centre"],
          ["justifyRight", "Align right", "Right"],
        ].map(([cmd, title]) => (
          <button
            key={cmd}
            type="button"
            disabled={disabled}
            title={title}
            aria-label={title}
            onMouseDown={(e) => e.preventDefault()}
            onClick={() => command(cmd)}
          >
            <ToolbarIcon name={cmd} />
          </button>
        ))}
        <button
          type="button"
          disabled={disabled}
          title="Code block"
          aria-label="Code block"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => command("formatBlock", "pre")}
        >
          <ToolbarIcon name="code" />
        </button>
        <select
          aria-label="Font size"
          disabled={disabled}
          defaultValue="16"
          onFocus={remember}
          onChange={(e) => {
            restore();
            document.execCommand("fontSize", false, "7");
            root.current?.querySelectorAll('font[size="7"]').forEach((font) => {
              (font as HTMLElement).style.fontSize = `${e.target.value}px`;
              font.removeAttribute("size");
            });
            changed();
          }}
        >
          {[12, 14, 16, 18, 20, 24].map((size) => (
            <option key={size} value={size}>
              {size}px
            </option>
          ))}
        </select>
        <input
          type="color"
          aria-label="Text colour"
          disabled={disabled}
          defaultValue="#008844"
          onFocus={remember}
          onChange={(e) => command("foreColor", e.target.value)}
        />
        <button
          type="button"
          disabled={disabled}
          onMouseDown={remember}
          title="Insert link"
          aria-label="Insert link"
          onClick={() => {
            setTool(tool === "link" ? null : "link");
            setError("");
          }}
        >
          <ToolbarIcon name="link" />
        </button>
        <button
          type="button"
          disabled={disabled}
          onMouseDown={remember}
          title="Add GIF"
          aria-label="Add GIF"
          onClick={() => {
            setTool(tool === "gif" ? null : "gif");
            setError("");
          }}
        >
          <ToolbarIcon name="gif" />
        </button>
      </div>
      {tool && (
        <div className={styles.controls}>
          <input
            aria-label={tool === "gif" ? "GIF URL" : "Link URL"}
            placeholder={
              tool === "gif" ? "https://… direct GIF URL" : "https://…"
            }
            value={url}
            onChange={(e) => setURL(e.target.value)}
            disabled={disabled}
            maxLength={2048}
          />
          <input
            aria-label={tool === "gif" ? "GIF description" : "Link label"}
            placeholder={
              tool === "gif"
                ? "Description (optional)"
                : "Link label (optional)"
            }
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={disabled}
            maxLength={tool === "gif" ? 200 : 2000}
          />
          <Button type="button" disabled={disabled} onClick={addURL}>
            Insert
          </Button>
          <Button type="button" onClick={() => setTool(null)}>
            Cancel
          </Button>
        </div>
      )}
      <div
        ref={root}
        className={styles.body}
        contentEditable={!disabled}
        suppressContentEditableWarning
        role="textbox"
        aria-multiline="true"
        aria-label={label}
        aria-autocomplete="list"
        aria-expanded={!!query}
        aria-controls={query ? suggestionId : undefined}
        aria-activedescendant={
          options[selected] ? `${suggestionId}-${selected}` : undefined
        }
        onInput={changed}
        onKeyUp={remember}
        onMouseUp={changed}
        onBlur={remember}
        onDrop={(e) => e.preventDefault()}
        onPaste={(e) => {
          e.preventDefault();
          if (disabled) return;
          remember();
          const sources = clipboardImageSources(
            e.clipboardData.getData("text/html"),
          );
          if (sources.length) {
            const count = root.current
              ? readEditor(root.current)
                  .blocks.flatMap((b) => b.children)
                  .filter((n) => n.type === "gif").length
              : 0;
            if (count + sources.length > 10) {
              setError("A comment can contain up to 10 GIFs.");
              return;
            }
            const fragment = document.createDocumentFragment();
            for (const src of sources) {
              const image = document.createElement("img");
              image.src = src;
              image.alt = "";
              image.className = styles.gif;
              image.referrerPolicy = "no-referrer";
              fragment.append(image);
            }
            insert(fragment);
            setError("");
            return;
          }
          const pasted = autolink({
            type: "text",
            text: e.clipboardData.getData("text/plain"),
          });
          const gifs = pasted.filter((node) => node.type === "gif");
          if (gifs.length) {
            const count = root.current
              ? readEditor(root.current)
                  .blocks.flatMap((b) => b.children)
                  .filter((node) => node.type === "gif").length
              : 0;
            if (count + gifs.length > 10) {
              setError("A comment can contain up to 10 GIFs.");
              return;
            }
            const fragment = document.createDocumentFragment();
            for (const node of pasted) {
              if (node.type === "gif") {
                const image = document.createElement("img");
                image.src = node.src!;
                image.alt = "";
                image.className = styles.gif;
                image.referrerPolicy = "no-referrer";
                fragment.append(image);
              } else fragment.append(document.createTextNode(node.text || ""));
            }
            insert(fragment);
            setError("");
            return;
          }
          if (
            [...e.clipboardData.items].some(
              (item) => item.kind === "file" && item.type.startsWith("image/"),
            )
          ) {
            setError(
              "Your browser copied image pixels without an address. Use “Copy image address” and Add GIF; file uploads are not supported yet.",
            );
            return;
          }
          command("insertText", e.clipboardData.getData("text/plain"));
        }}
        onKeyDown={(e) => {
          if (
            options.length &&
            ["ArrowDown", "ArrowUp", "Enter"].includes(e.key)
          ) {
            e.preventDefault();
            if (e.key === "Enter") mention(options[selected]);
            else
              setSelected(
                (i) =>
                  (i + (e.key === "ArrowDown" ? 1 : options.length - 1)) %
                  options.length,
              );
          }
          if (e.key === "Escape") {
            setQuery("");
            setOptions([]);
          }
        }}
      />
      {query && (
        <div
          id={suggestionId}
          className={styles.suggestions}
          role="listbox"
          aria-label="Mention suggestions"
        >
          {searching ? (
            <span>Searching people…</span>
          ) : query.length < 2 ? (
            <span>Type at least two letters after @.</span>
          ) : !options.length ? (
            <span>No matching active users.</span>
          ) : (
            options.map((person, i) => (
              <button
                id={`${suggestionId}-${i}`}
                type="button"
                role="option"
                aria-selected={selected === i}
                key={person.id}
                onMouseDown={(e) => e.preventDefault()}
                onClick={() => mention(person)}
              >
                {person.name}
              </button>
            ))
          )}
        </div>
      )}
      {error && (
        <div className={styles.error} role="alert">
          {error}
        </div>
      )}
      <div className={styles.help}>
        Type @name to mention someone. Typed web links become clickable when
        posted. Direct HTTPS .gif links become GIFs automatically. You can also
        paste a copied GIF or use Add GIF.
      </div>
    </div>
  );
}
