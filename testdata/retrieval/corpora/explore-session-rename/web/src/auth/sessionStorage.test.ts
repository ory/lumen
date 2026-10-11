import { describe, expect, it } from 'vitest';
import { decodeSession } from '../api/session';
import { currentSessionId, rememberSession } from './sessionStorage';

describe('session storage', () => {
  it('remembers the decoded sessionId', () => {
    const session = decodeSession({ session_id: 's-123', user_id: 'u-1', expires_at: '2030-01-01T00:00:00Z' });
    rememberSession(session);
    expect(currentSessionId()).toBe('s-123');
  });
});
