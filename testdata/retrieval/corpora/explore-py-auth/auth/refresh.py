import time

from auth.errors import RefreshTokenReusedError

_used: set[str] = set()


def rotate_refresh_token(refresh_id: str, issued_at: float, ttl: int) -> None:
    if refresh_id in _used:
        raise RefreshTokenReusedError("refresh token was already used")
    if issued_at + ttl < time.time():
        raise RefreshTokenReusedError("refresh token is too old to rotate")
    _used.add(refresh_id)
