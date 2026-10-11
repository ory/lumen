import type { Clock } from '../platform/clock';
import type { User, UserRepository } from './UserRepository';

export class UserNotFoundError extends Error {
  constructor(readonly userId: string) {
    super(`user ${userId} not found`);
  }
}

export class UserService {
  constructor(
    private readonly users: UserRepository,
    private readonly clock: Clock,
  ) {}

  async getUser(id: string): Promise<User> {
    const user = await this.users.findById(id);
    if (!user) {
      throw new UserNotFoundError(id);
    }
    return user;
  }

  async disableUser(id: string): Promise<void> {
    const user = await this.getUser(id);
    if (user.disabledAt) {
      return;
    }
    await this.users.save({ ...user, disabledAt: this.clock.now() });
  }
}
