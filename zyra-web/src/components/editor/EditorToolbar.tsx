import { ToolbarIcon } from "./ToolbarIcon";
import styles from "./RichEditor.module.css";
export function EditorToolbar({
  disabled,
  command,
  remember,
  onFontSize,
  onTool,
}: {
  disabled: boolean;
  command: (name: string, value?: string) => void;
  remember: () => void;
  onFontSize: (size: string) => void;
  onTool: (tool: "link" | "gif") => void;
}) {
  return (
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
        onChange={(e) => onFontSize(e.target.value)}
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
        onClick={() => onTool("link")}
      >
        <ToolbarIcon name="link" />
      </button>
      <button
        type="button"
        disabled={disabled}
        onMouseDown={remember}
        title="Add GIF"
        aria-label="Add GIF"
        onClick={() => onTool("gif")}
      >
        <ToolbarIcon name="gif" />
      </button>
    </div>
  );
}
