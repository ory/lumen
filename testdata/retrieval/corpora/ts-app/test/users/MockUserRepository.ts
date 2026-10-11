import type { User, UserRepository } from '../../src/users/UserRepository';

export class MockUserRepository implements UserRepository {
  readonly saved: User[] = [];
  private readonly byId = new Map<string, User>();

  constructor(seed: User[] = []) {
    for (const user of seed) {
      this.byId.set(user.id, user);
    }
  }

  async findById(id: string): Promise<User | undefined> {
    return this.byId.get(id);
  }

  async findByEmail(email: string): Promise<User | undefined> {
    return [...this.byId.values()].find((u) => u.email === email);
  }

  async save(user: User): Promise<void> {
    this.saved.push(user);
    this.byId.set(user.id, user);
  }
}
