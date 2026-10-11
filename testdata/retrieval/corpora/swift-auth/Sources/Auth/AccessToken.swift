import Foundation

/// A bearer credential issued by the identity provider.
public struct AccessToken: Codable, Sendable, Equatable {
    public let value: String
    public let refreshToken: String
    public let expiresAt: Date

    public var isExpired: Bool {
        expiresAt.timeIntervalSinceNow < 30
    }
}
