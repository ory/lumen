import { describe, expect, it } from 'vitest';
import { UserNotFoundError, UserService } from '../../src/users/UserService';
import { MockUserRepository } from './MockUserRepository';

const fixedClock = { now: () => new Date('2026-01-01T00:00:00Z') };

describe('UserService', () => {
  it('disables an active user with the injected repository', async () => {
    const repository = new MockUserRepository([{ id: 'u1', email: 'a@example.com', displayName: 'A' }]);
    const service = new UserService(repository, fixedClock);

    await service.disableUser('u1');

    expect(repository.saved[0].disabledAt).toEqual(fixedClock.now());
  });

  it('throws when the user is missing', async () => {
    const service = new UserService(new MockUserRepository(), fixedClock);
    await expect(service.getUser('missing')).rejects.toBeInstanceOf(UserNotFoundError);
  });
});
