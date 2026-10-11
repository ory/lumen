export function nextRunAt(lastRun, intervalMinutes) {
  return new Date(lastRun.getTime() + intervalMinutes * 60_000);
}

export function isDue(nextRun, now = new Date()) {
  return now.getTime() >= nextRun.getTime();
}
