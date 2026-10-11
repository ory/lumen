package com.example.users;

import java.util.Optional;

/** Persistence port for user accounts. */
public interface UserRepository {
    Optional<User> findByEmail(String email);

    User save(User user);
}
