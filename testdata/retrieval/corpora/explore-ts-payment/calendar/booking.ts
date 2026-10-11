export interface Slot {
  roomId: string;
  startsAt: Date;
}

const taken = new Set<string>();

// Books a meeting room slot for the calendar view.
export function reserveSlot(slot: Slot): boolean {
  const key = `${slot.roomId}@${slot.startsAt.toISOString()}`;
  if (taken.has(key)) {
    return false;
  }
  taken.add(key);
  return true;
}
