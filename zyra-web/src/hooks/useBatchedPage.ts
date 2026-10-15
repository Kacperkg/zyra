import { useCallback, useEffect, useRef, useState } from "react";
import type { Page } from "../types/api";

// A displayed page contains at most two 50-item API pages.
export function useBatchedPage<T extends { id: string }>({
  identity,
  page,
  fetchPage,
  observe = true,
}: {
  identity: string;
  page: number;
  fetchPage: (page: number, signal: AbortSignal) => Promise<Page<T>>;
  observe?: boolean;
}) {
  const [items, setItems] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [loaded, setLoaded] = useState(0);
  const [secondLoaded, setSecondLoaded] = useState(false);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  const sentinel = useRef<HTMLDivElement>(null);
  const fetchRef = useRef(fetchPage);
  fetchRef.current = fetchPage;
  const currentRequest = useRef<AbortController | null>(null);
  const fetching = useRef(false);

  useEffect(() => {
    const controller = new AbortController();
    currentRequest.current = controller;
    fetching.current = true;
    setItems([]);
    setTotal(0);
    setLoaded(0);
    setSecondLoaded(false);
    setBusy(true);
    setError("");
    fetchRef
      .current((page - 1) * 2 + 1, controller.signal)
      .then((data) => {
        if (controller.signal.aborted) return;
        setItems(data.items);
        setTotal(data.total);
        setLoaded(data.items.length);
      })
      .catch((e) => {
        if (!controller.signal.aborted) setError(e.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) {
          fetching.current = false;
          setBusy(false);
        }
      });
    return () => controller.abort();
  }, [identity, page]);

  const more = useCallback(async () => {
    const controller = currentRequest.current;
    if (
      !controller ||
      controller.signal.aborted ||
      fetching.current ||
      secondLoaded ||
      loaded !== 50 ||
      (page - 1) * 100 + loaded >= total
    )
      return;
    fetching.current = true;
    setBusy(true);
    setError("");
    try {
      const data = await fetchRef.current(
        (page - 1) * 2 + 2,
        controller.signal,
      );
      if (controller.signal.aborted) return;
      setItems((old) => {
        const ids = new Set(old.map((item) => item.id));
        return [
          ...old,
          ...data.items.filter((item) => {
            if (ids.has(item.id)) return false;
            ids.add(item.id);
            return true;
          }),
        ];
      });
      setLoaded(50 + data.items.length);
      setSecondLoaded(true);
      setTotal(data.total);
    } catch (e) {
      if (!controller.signal.aborted) setError((e as Error).message);
    } finally {
      if (!controller.signal.aborted) {
        fetching.current = false;
        setBusy(false);
      }
    }
  }, [page, loaded, total, secondLoaded]);

  useEffect(() => {
    const node = sentinel.current;
    if (!node || !observe || busy || error) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) void more();
      },
      { rootMargin: "120px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [observe, busy, error, more]);

  return { items, total, loaded, secondLoaded, busy, error, sentinel };
}
