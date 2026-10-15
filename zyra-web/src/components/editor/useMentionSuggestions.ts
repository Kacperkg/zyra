import { useEffect, useState } from "react";
import { mentionOptions, type MentionOption } from "../../api/mentions.api";
export function useMentionSuggestions(
  query: string,
  setError: (message: string) => void,
) {
  const [options, setOptions] = useState<MentionOption[]>([]);
  const [selected, setSelected] = useState(0);
  const [searching, setSearching] = useState(false);
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
  }, [query, setError]);

  return { options, setOptions, selected, setSelected, searching };
}
