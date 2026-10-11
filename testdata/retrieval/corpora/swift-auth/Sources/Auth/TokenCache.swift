import Foundation

/// In-memory cache used by views that only need to display the last known token.
public final class TokenCache {
    private var entries: [String: AccessToken] = [:]
    private let lock = NSLock()

    public init() {}

    /// Returns the cached token without contacting the identity provider.
    public func token(for account: String) -> AccessToken? {
        lock.lock()
        defer { lock.unlock() }
        return entries[account]
    }

    public func store(_ token: AccessToken, for account: String) {
        lock.lock()
        defer { lock.unlock() }
        entries[account] = token
    }

    public func evictExpired() {
        lock.lock()
        defer { lock.unlock() }
        entries = entries.filter { !$0.value.isExpired }
    }
}
