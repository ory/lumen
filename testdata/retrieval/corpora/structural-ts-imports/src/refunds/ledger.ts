import { db } from '../db';

export async function reverseCharge(chargeId: string): Promise<boolean> {
  const res = await db.query('UPDATE charges SET reversed_at = now() WHERE id = $1 AND reversed_at IS NULL', [chargeId]);
  return res.rowCount === 1;
}
