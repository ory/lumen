import type { Pool } from 'pg';
import type { User, UserRepository } from './UserRepository';

export class PostgresUserRepository implements UserRepository {
  constructor(private readonly pool: Pool) {}

  async findById(id: string): Promise<User | undefined> {
    const { rows } = await this.pool.query('SELECT id, email, display_name, disabled_at FROM users WHERE id = $1', [id]);
    return rows[0] ? toUser(rows[0]) : undefined;
  }

  async findByEmail(email: string): Promise<User | undefined> {
    const { rows } = await this.pool.query('SELECT id, email, display_name, disabled_at FROM users WHERE email = $1', [email]);
    return rows[0] ? toUser(rows[0]) : undefined;
  }

  async save(user: User): Promise<void> {
    await this.pool.query(
      'INSERT INTO users (id, email, display_name, disabled_at) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET email = $2, display_name = $3, disabled_at = $4',
      [user.id, user.email, user.displayName, user.disabledAt ?? null],
    );
  }
}

function toUser(row: { id: string; email: string; display_name: string; disabled_at: Date | null }): User {
  return { id: row.id, email: row.email, displayName: row.display_name, disabledAt: row.disabled_at ?? undefined };
}
