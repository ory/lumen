export interface Session {
  sessionId: string;
  userId: string;
  expiresAt: Date;
}

interface SessionResponse {
  session_id: string;
  user_id: string;
  expires_at: string;
}

export function decodeSession(raw: SessionResponse): Session {
  return {
    sessionId: raw.session_id,
    userId: raw.user_id,
    expiresAt: new Date(raw.expires_at),
  };
}

export async function login(email: string, password: string): Promise<Session> {
  const res = await fetch('/api/login', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) {
    throw new Error(`login failed: ${res.status}`);
  }
  return decodeSession(await res.json());
}
