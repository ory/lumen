import time

from auth.errors import CredentialExpiredError
from auth.jwt_codec import decode_token

CLOCK_SKEW_SECONDS = 30


def validate_token(token: str, secret: bytes, now: float | None = None) -> dict:
    claims = decode_token(token, secret)
    current = now if now is not None else time.time()
    if claims["exp"] + CLOCK_SKEW_SECONDS < current:
        raise CredentialExpiredError("Your session credential has expired. Please sign in again.")
    return claims
