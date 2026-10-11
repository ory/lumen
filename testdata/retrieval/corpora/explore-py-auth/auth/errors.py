class AuthError(Exception):
    """Base class for authentication failures returned as HTTP 401."""

    status_code = 401


class CredentialExpiredError(AuthError):
    pass


class InvalidSignatureError(AuthError):
    pass


class RefreshTokenReusedError(AuthError):
    status_code = 403
