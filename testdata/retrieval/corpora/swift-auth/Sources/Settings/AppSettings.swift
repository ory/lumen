import Foundation

/// User-facing preferences shown on the settings screen.
public struct AppSettings {
    @Persisted(key: "settings.theme", defaultValue: "system")
    public var theme: String

    @Persisted(key: "settings.notificationsEnabled", defaultValue: true)
    public var notificationsEnabled: Bool

    @Persisted(key: "settings.lastSyncedAccount", defaultValue: nil)
    public var lastSyncedAccount: String?

    public init() {}

    public mutating func resetTheme() {
        theme = "system"
    }
}
