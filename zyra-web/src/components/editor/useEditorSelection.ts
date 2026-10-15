import { useRef, type RefObject } from "react";
export function useEditorSelection(
  root: RefObject<HTMLDivElement | null>,
  onInserted: () => void,
) {
  const selection = useRef<Range | null>(null);
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
    onInserted();
  }

  return { selection, remember, restore, insert };
}
