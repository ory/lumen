import type { Session } from '../api/session';

const KEY = 'app.session';

export function rememberSession(session: Session): void {
  localStorage.setItem(KEY, JSON.stringify({ ...session, expiresAt: session.expiresAt.toISOString() }));
}

export function currentSessionId(): string | null {
  const stored = localStorage.getItem(KEY);
  return stored ? (JSON.parse(stored).sessionId as string) : null;
}
