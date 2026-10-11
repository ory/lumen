import { drainQueue } from './queue.mjs';

export function flushPendingJobs(log: (line: string) => void): number {
  const jobs = drainQueue();
  for (const job of jobs) {
    log(`flushing ${job}`);
  }
  return jobs.length;
}
