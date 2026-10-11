import { Pool } from 'pg';

const pool = new Pool();

export class Ledger {
  async insertIfAbsent(key: string, at: number): Promise<boolean> {
    const res = await pool.query(
      'INSERT INTO idempotency_keys (key, created_at) VALUES ($1, to_timestamp($2 / 1000.0)) ON CONFLICT (key) DO NOTHING',
      [key, at],
    );
    return res.rowCount === 1;
  }

  async remove(key: string): Promise<void> {
    await pool.query('DELETE FROM idempotency_keys WHERE key = $1', [key]);
  }
}

export const ledger = new Ledger();
