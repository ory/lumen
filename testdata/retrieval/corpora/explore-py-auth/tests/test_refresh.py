import time

import pytest

from auth.errors import RefreshTokenReusedError
from auth.refresh import rotate_refresh_token


def test_expired_refresh_token_cannot_rotate():
    with pytest.raises(RefreshTokenReusedError):
        rotate_refresh_token("r1", issued_at=time.time() - 10_000, ttl=60)
