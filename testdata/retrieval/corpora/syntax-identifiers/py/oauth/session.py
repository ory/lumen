class BrowserSession:
    """Interactive login session backed by a refresh cookie."""

    def __init__(self, http):
        self.http = http
        self.access = None

    def refresh_token(self):
        response = self.http.post("/oauth/token", data={"grant_type": "refresh_token"})
        self.access = response.json()["access_token"]
        return self.access
