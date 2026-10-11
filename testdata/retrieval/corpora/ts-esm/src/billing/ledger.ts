const entries = new Set<string>();

export const ledger = {
  insertIfAbsent(key: string): boolean {
    if (entries.has(key)) {
      return false;
    }
    entries.add(key);
    return true;
  },
  remove(key: string): void {
    entries.delete(key);
  },
};

function releaseHold(holdId: string): void {
  ledger.remove(`slot:${holdId}`);
}

function holdCount(): number {
  return entries.size;
}

export { releaseHold, holdCount };
