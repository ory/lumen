package com.example.users;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class SignupServiceTest {
    private final InMemoryUserRepository users = new InMemoryUserRepository();

    @Test
    void findByEmailAfterSignup() {
        users.save(new User(0, "grace@example.com", "Grace"));

        assertThat(users.findByEmail("grace@example.com")).isPresent();
    }
}
