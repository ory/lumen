import Foundation

/// Serializes token refreshes so concurrent callers share one network round trip.
public actor TokenStore {
    private let client: AuthClient
    private var tokens: [String: AccessToken] = [:]
    private var inFlight: [String: Task<AccessToken, Error>] = [:]

    public init(client: AuthClient) {
        self.client = client
    }

    public func seed(_ token: AccessToken, for account: String) {
        tokens[account] = token
    }

    /// Returns a valid token, refreshing it with the identity provider when it expired.
    public func refreshToken(for account: String) async throws -> AccessToken {
        if let current = tokens[account], !current.isExpired {
            return current
        }
        if let pending = inFlight[account] {
            return try await pending.value
        }
        guard let stale = tokens[account] else {
            throw AuthError.missingCredentials(account: account)
        }

        let task = Task { try await client.exchange(refreshToken: stale.refreshToken) }
        inFlight[account] = task
        defer { inFlight[account] = nil }

        let fresh = try await task.value
        tokens[account] = fresh
        return fresh
    }
}
