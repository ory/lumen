const pending: string[] = [];

export function enqueueJob(name: string): number {
  pending.push(name);
  return pending.length;
}

export function drainQueue(): string[] {
  return pending.splice(0, pending.length);
}
