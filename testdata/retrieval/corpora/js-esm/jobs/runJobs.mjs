import { nextRunAt } from './schedule.mjs';

export function rescheduleJob(job, now = new Date()) {
  return { ...job, lastRun: now, nextRun: nextRunAt(now, job.intervalMinutes) };
}
