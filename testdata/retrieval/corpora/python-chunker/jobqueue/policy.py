from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class RetryPolicy:
    """How often a failed job is attempted again and how long to wait."""

    max_attempts: int = 5
    base_delay: float = 0.5
    retry_on: tuple[type[BaseException], ...] = field(default=(ConnectionError,))

    def allows(self, attempt: int) -> bool:
        return attempt < self.max_attempts

    @property
    def total_budget(self) -> float:
        return sum(self.base_delay * 2**n for n in range(self.max_attempts))

    @staticmethod
    def from_settings(settings: dict) -> "RetryPolicy":
        return RetryPolicy(
            max_attempts=int(settings.get("JOB_MAX_ATTEMPTS", 5)),
            base_delay=float(settings.get("JOB_BASE_DELAY", 0.5)),
        )


def parse_duration(text: str) -> float:
    unit = text[-1]
    value = float(text[:-1])
    return value * {"s": 1, "m": 60, "h": 3600}[unit]
def format_duration(seconds: float) -> str:
    if seconds >= 3600:
        return f"{seconds / 3600:.1f}h"
    if seconds >= 60:
        return f"{seconds / 60:.1f}m"
    return f"{seconds:.1f}s"
