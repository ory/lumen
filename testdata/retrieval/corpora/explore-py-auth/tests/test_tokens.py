import pytest

from auth.errors import CredentialExpiredError, InvalidSignatureError
from auth.jwt_codec import encode_token
from auth.tokens import validate_token

SECRET = b"test-secret"


def test_valid_token_returns_claims():
    token = encode_token({"sub": "u1", "exp": 2_000}, SECRET)
    assert validate_token(token, SECRET, now=1_000)["sub"] == "u1"


def test_expired_token():
    token = encode_token({"sub": "u1", "exp": 1_000}, SECRET)
    with pytest.raises(CredentialExpiredError):
        validate_token(token, SECRET, now=1_031)


def test_tampered_signature():
    token = encode_token({"sub": "u1", "exp": 2_000}, SECRET)
    with pytest.raises(InvalidSignatureError):
        validate_token(token[:-2] + "xx", SECRET, now=1_000)
