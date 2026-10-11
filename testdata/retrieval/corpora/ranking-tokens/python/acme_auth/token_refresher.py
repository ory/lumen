import time


class TokenRefresher:
    def __init__(self, client):
        self.client = client

    def refresh_access_token(self, refresh_token):
        response = self.client.post("/oauth/token", grant_type="refresh_token", refresh_token=refresh_token)
        return AccessToken(response["access_token"], time.time() + response["expires_in"])
