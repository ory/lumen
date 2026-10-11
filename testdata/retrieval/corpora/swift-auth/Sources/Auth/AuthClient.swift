import Foundation

public enum AuthError: Error {
    case missingCredentials(account: String)
    case invalidResponse(status: Int)
}

/// Talks to the OAuth token endpoint.
public struct AuthClient: Sendable {
    let session: URLSession
    let tokenURL: URL

    public init(session: URLSession = .shared, tokenURL: URL) {
        self.session = session
        self.tokenURL = tokenURL
    }

    /// Exchanges a refresh token for a new access token.
    public func exchange(refreshToken: String) async throws -> AccessToken {
        var request = URLRequest(url: tokenURL)
        request.httpMethod = "POST"
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        request.httpBody = Data("grant_type=refresh_token&refresh_token=\(refreshToken)".utf8)

        let (data, response) = try await session.data(for: request)
        guard let http = response as? HTTPURLResponse, http.statusCode == 200 else {
            throw AuthError.invalidResponse(status: (response as? HTTPURLResponse)?.statusCode ?? 0)
        }
        return try JSONDecoder().decode(AccessToken.self, from: data)
    }
}
