import { rooms } from '../db';

export async function holdRoom(roomId: string, nights: number): Promise<void> {
  await rooms.update(roomId, { heldUntil: Date.now() + 15 * 60_000, nights });
}
