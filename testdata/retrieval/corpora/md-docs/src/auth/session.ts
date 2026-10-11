export interface Session {
	accessToken: string;
	refreshToken: string;
	expiresAt: number;
}

export interface SessionStore {
	load(): Promise<Session>;
	save(session: Session): Promise<void>;
}

export function isExpired(session: Session, now = Date.now()): boolean {
	return session.expiresAt <= now;
}
