package com.acme.auth;

import java.time.Instant;

public final class TokenRefresher {
  private final OAuthClient client;

  public TokenRefresher(OAuthClient client) {
    this.client = client;
  }

  public AccessToken refreshAccessToken(RefreshToken refreshToken) {
    TokenResponse response = client.post("/oauth/token", "refresh_token", refreshToken.value());
    return new AccessToken(response.accessToken(), Instant.now().plusSeconds(response.expiresIn()));
  }
}
