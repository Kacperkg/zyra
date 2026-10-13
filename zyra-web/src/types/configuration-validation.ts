import type { ResourceRule } from "./configuration";
export function optionalNumber(
  value: string,
  kind: "percentage" | "sequences",
): number | null {
  if (!value.trim()) return null;
  const number = Number(value);
  if (
    !Number.isFinite(number) ||
    number < 0 ||
    (kind === "percentage"
      ? number > 100
      : !Number.isInteger(number) || number > 2147483647)
  )
    throw new Error(
      kind === "percentage"
        ? "Percentage must be between 0 and 100."
        : "Lag must be a whole number between 0 and 2147483647 sequences.",
    );
  return number;
}
export function resourceRule(
  ignore: boolean,
  threshold: string,
  kind: "max_used_percent" | "min_free_percent" | null,
): ResourceRule {
  const value = kind ? optionalNumber(threshold, "percentage") : null;
  return { ignore, ...(kind && value !== null ? { [kind]: value } : {}) };
}
export function receivedTimestamp(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unavailable" : date.toLocaleString();
}
