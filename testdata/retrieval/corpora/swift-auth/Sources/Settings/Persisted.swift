import Foundation

/// Stores a Codable value in UserDefaults under a fixed key.
@propertyWrapper
public struct Persisted<Value: Codable> {
    private let key: String
    private let defaultValue: Value
    private let defaults: UserDefaults

    public init(key: String, defaultValue: Value, defaults: UserDefaults = .standard) {
        self.key = key
        self.defaultValue = defaultValue
        self.defaults = defaults
    }

    public var wrappedValue: Value {
        get {
            guard let data = defaults.data(forKey: key),
                  let value = try? JSONDecoder().decode(Value.self, from: data) else {
                return defaultValue
            }
            return value
        }
        set {
            let data = try? JSONEncoder().encode(newValue)
            defaults.set(data, forKey: key)
        }
    }
}
