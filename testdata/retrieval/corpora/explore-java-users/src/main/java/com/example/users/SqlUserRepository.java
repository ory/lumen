package com.example.users;

import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.Optional;
import javax.sql.DataSource;

public class SqlUserRepository implements UserRepository {
    private final DataSource dataSource;

    public SqlUserRepository(DataSource dataSource) {
        this.dataSource = dataSource;
    }

    @Override
    public Optional<User> findByEmail(String email) {
        String sql = "SELECT id, email, display_name FROM users WHERE lower(email) = lower(?)";
        try (Connection conn = dataSource.getConnection();
                PreparedStatement stmt = conn.prepareStatement(sql)) {
            stmt.setString(1, email);
            try (ResultSet rs = stmt.executeQuery()) {
                if (!rs.next()) {
                    return Optional.empty();
                }
                return Optional.of(new User(rs.getLong("id"), rs.getString("email"), rs.getString("display_name")));
            }
        } catch (SQLException e) {
            throw new RepositoryException("findByEmail failed", e);
        }
    }

    @Override
    public User save(User user) {
        String sql = "INSERT INTO users (email, display_name) VALUES (?, ?) RETURNING id";
        try (Connection conn = dataSource.getConnection();
                PreparedStatement stmt = conn.prepareStatement(sql)) {
            stmt.setString(1, user.email());
            stmt.setString(2, user.displayName());
            try (ResultSet rs = stmt.executeQuery()) {
                rs.next();
                return new User(rs.getLong(1), user.email(), user.displayName());
            }
        } catch (SQLException e) {
            throw new RepositoryException("save failed", e);
        }
    }
}
