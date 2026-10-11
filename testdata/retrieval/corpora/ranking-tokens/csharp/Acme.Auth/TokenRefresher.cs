using System;
using System.Threading.Tasks;

namespace Acme.Auth
{
    public sealed class TokenRefresher
    {
        private readonly OAuthClient _client;

        public TokenRefresher(OAuthClient client) => _client = client;

        public async Task<AccessToken> RefreshAccessToken(RefreshToken refreshToken)
        {
            var response = await _client.PostAsync("/oauth/token", "refresh_token", refreshToken.Value);
            return new AccessToken(response.AccessToken, DateTimeOffset.UtcNow.AddSeconds(response.ExpiresIn));
        }
    }
}
