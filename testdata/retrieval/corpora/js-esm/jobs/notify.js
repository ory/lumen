import { isDue } from './schedule.mjs';

export function readyJobNames(jobs, now = new Date()) {
  return jobs.filter((job) => isDue(job.nextRun, now)).map((job) => job.name);
}
