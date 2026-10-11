from __future__ import annotations

import functools
import random
import time

from .policy import RetryPolicy

# def backoff_delay_v1(attempt):
#     return 2 ** attempt


def backoff_delay(attempt: int, base: float = 0.5, jitter: bool = True) -> float:
    delay = base * 2**attempt
    if jitter:
        delay += random.uniform(0, base)
    return delay


def retry(policy: RetryPolicy | None = None):
    policy = policy or RetryPolicy()

    def decorator(func):
        @functools.wraps(func)
        def wrapper(*args, **kwargs):
            attempt = 0
            while True:
                try:
                    return func(*args, **kwargs)
                except policy.retry_on:
                    attempt += 1
                    if not policy.allows(attempt):
                        raise
                    time.sleep(backoff_delay(attempt, policy.base_delay))

        return wrapper

    return decorator


async def fetch_with_retry(client, url: str, policy: RetryPolicy | None = None):
    policy = policy or RetryPolicy()
    for attempt in range(policy.max_attempts):
        try:
            return await client.get(url)
        except policy.retry_on:
            if not policy.allows(attempt + 1):
                raise
    return None
