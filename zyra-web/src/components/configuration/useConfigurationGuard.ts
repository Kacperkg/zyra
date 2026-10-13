import { useBlocker } from "@tanstack/react-router";
export function useConfigurationGuard(
  dirty: boolean,
  busy = false,
  bypass?: () => boolean,
) {
  useBlocker({
    shouldBlockFn: () =>
      !bypass?.() &&
      (busy ||
        (dirty && !window.confirm("Discard unsaved configuration changes?"))),
    enableBeforeUnload: dirty || busy,
  });
}
