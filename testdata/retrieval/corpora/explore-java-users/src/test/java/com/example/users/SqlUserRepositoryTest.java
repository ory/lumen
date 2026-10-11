package com.example.users;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

class SqlUserRepositoryTest {
    private SqlUserRepository repository;

    @BeforeEach
    void setUp() {
        repository = new SqlUserRepository(TestDatabase.dataSource());
        TestDatabase.truncate("users");
    }

    @Test
    void findByEmailIgnoresCase() {
        repository.save(new User(0, "Ada@Example.com", "Ada"));

        assertThat(repository.findByEmail("ada@example.COM"))
                .map(User::displayName)
                .contains("Ada");
    }

    @Test
    void findByEmailReturnsEmptyForUnknownEmail() {
        assertThat(repository.findByEmail("nobody@example.com")).isEmpty();
    }
}
