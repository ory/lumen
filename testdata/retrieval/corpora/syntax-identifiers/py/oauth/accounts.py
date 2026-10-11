import time


class ServiceAccount:
    """Machine identity that renews its own credentials."""

    def __init__(self, key_id, private_key):
        self.key_id = key_id
        self.private_key = private_key
        self.token = None

    def __refresh_token__(self, client):
        """Hook invoked by TokenManager when the cached token is stale."""
        assertion = client.sign_jwt(self.key_id, self.private_key, now=time.time())
        self.token = client.exchange_assertion(assertion)
        return self.token
