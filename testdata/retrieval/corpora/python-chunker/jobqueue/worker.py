from __future__ import annotations

import logging

from .policy import RetryPolicy
from .retry import backoff_delay

logger = logging.getLogger(__name__)

DEFAULT_QUEUE = "default"


class Worker:
    """Pulls jobs from a queue and runs them with retries."""

    class Stats:
        def __init__(self) -> None:
            self.processed = 0
            self.failed = 0

        def failure_rate(self) -> float:
            total = self.processed + self.failed
            return self.failed / total if total else 0.0

    def __init__(self, queue, policy: RetryPolicy | None = None) -> None:
        self.queue = queue
        self.policy = policy or RetryPolicy()
        self.stats = Worker.Stats()

    def run_once(self) -> bool:
        job = self.queue.pop(DEFAULT_QUEUE)
        if job is None:
            return False
        try:
            job.perform()
        except self.policy.retry_on as exc:
            self.stats.failed += 1
            if self.policy.allows(job.attempts + 1):
                delay = backoff_delay(job.attempts, self.policy.base_delay)
                self.queue.release(job, delay=delay)
            else:
                logger.error("job %s exhausted retries: %s", job.id, exc)
                self.queue.bury(job)
        else:
            self.stats.processed += 1
        return True
