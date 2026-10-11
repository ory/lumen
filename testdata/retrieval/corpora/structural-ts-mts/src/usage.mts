export function summarizeUsage(events: { kind: string; units: number }[]): { totalUnits: number; kinds: number } {
  const kinds = new Set(events.map((e) => e.kind));
  return { totalUnits: events.reduce((n, e) => n + e.units, 0), kinds: kinds.size };
}
