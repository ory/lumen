package com.example.users;

import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

/** Test fake; compares emails exactly. */
class InMemoryUserRepository implements UserRepository {
    private final Map<String, User> byEmail = new HashMap<>();
    private long nextId = 1;

    @Override
    public Optional<User> findByEmail(String email) {
        return Optional.ofNullable(byEmail.get(email));
    }

    @Override
    public User save(User user) {
        User stored = new User(nextId++, user.email(), user.displayName());
        byEmail.put(stored.email(), stored);
        return stored;
    }
}
