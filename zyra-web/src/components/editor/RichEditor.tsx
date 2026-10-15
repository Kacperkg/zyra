import { useEffect, useId, useRef, useState } from "react";
import { type MentionOption } from "../../api/mentions.api";
import { Button } from "../ui/Controls";
import {
  emptyContent,
  readEditor,
  safeURL,
  autolink,
  tenorPageURL,
  type RichContent,
} from "./content";
import styles from "./RichEditor.module.css";
import { EditorToolbar } from "./EditorToolbar";
import { clipboardImageSources } from "./clipboard";
import { populateEditor, createGif, hasGifCapacity } from "./editor-dom";
import { useMentionSuggestions } from "./useMentionSuggestions";
import { useEditorSelection } from "./useEditorSelection";

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
  const { selection, remember, restore, insert } = useEditorSelection(
    root,
    changed,
  );
  const mentionRange = useRef<Range | null>(null);
  const [tool, setTool] = useState<"link" | "gif" | null>(null);
  const [url, setURL] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const { options, setOptions, selected, setSelected, searching } =
    useMentionSuggestions(query, setError);
  const initial = useRef(initialContent || emptyContent());
  useEffect(() => {
    if (root.current) populateEditor(root.current, initial.current, names);
  }, []);
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
      if (!hasGifCapacity(root.current, 1)) {
        setError("A comment can contain up to 10 GIFs.");
        return;
      }
      insert(createGif(href, description));
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
      <EditorToolbar
        disabled={disabled}
        command={command}
        remember={remember}
        onFontSize={(size) => {
          restore();
          document.execCommand("fontSize", false, "7");
          root.current?.querySelectorAll('font[size="7"]').forEach((font) => {
            (font as HTMLElement).style.fontSize = `${size}px`;
            font.removeAttribute("size");
          });
          changed();
        }}
        onTool={(next) => {
          setTool(tool === next ? null : next);
          setError("");
        }}
      />
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
            if (!hasGifCapacity(root.current, sources.length)) {
              setError("A comment can contain up to 10 GIFs.");
              return;
            }
            const fragment = document.createDocumentFragment();
            for (const src of sources) {
              fragment.append(createGif(src));
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
            if (!hasGifCapacity(root.current, gifs.length)) {
              setError("A comment can contain up to 10 GIFs.");
              return;
            }
            const fragment = document.createDocumentFragment();
            for (const node of pasted) {
              if (node.type === "gif") {
                fragment.append(createGif(node.src!));
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
