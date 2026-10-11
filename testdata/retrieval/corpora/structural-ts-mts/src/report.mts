import { summarizeUsage } from './usage.mjs';

export function buildMonthlyReport(accountId: string, events: { kind: string; units: number }[]): string {
  const summary = summarizeUsage(events);
  return `account ${accountId}: ${summary.totalUnits} units across ${summary.kinds} event kinds`;
}
