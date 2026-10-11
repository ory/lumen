import random
import time

MAX_ATTEMPTS = 5
BASE_DELAY = 0.5
MAX_DELAY = 30.0


class DeliveryFailed(Exception):
    """Raised when a webhook could not be delivered within the attempt budget."""


def deliver_once(endpoint, payload, send):
    """Send a webhook payload a single time and return the HTTP response."""
    return send(endpoint, payload)


def retry_delivery(endpoint, payload, send, max_attempts=MAX_ATTEMPTS, sleep=time.sleep):
    """Deliver a webhook payload, retrying with capped exponential backoff.

    Gives up after max_attempts and raises DeliveryFailed so the event can be
    parked in the dead-letter queue.
    """
    for attempt in range(1, max_attempts + 1):
        response = send(endpoint, payload)
        if 200 <= response.status < 300:
            return response
        if attempt == max_attempts:
            break
        delay = min(MAX_DELAY, BASE_DELAY * 2 ** (attempt - 1))
        sleep(delay + random.uniform(0, delay / 2))
    raise DeliveryFailed(f"{endpoint}: gave up after {max_attempts} attempts")
