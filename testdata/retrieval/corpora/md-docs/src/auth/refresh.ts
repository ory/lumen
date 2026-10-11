import type { Session, SessionStore } from "./session";

export interface RefreshOptions {
	force?: boolean;
}

const EXPIRY_SKEW_MS = 30_000;

export async function refreshAccessToken(store: SessionStore, options: RefreshOptions = {}): Promise<Session> {
	const current = await store.load();
	if (!options.force && current.expiresAt - Date.now() > EXPIRY_SKEW_MS) {
		return current;
	}
	const res = await fetch("/oauth/token", {
		method: "POST",
		headers: { "Content-Type": "application/x-www-form-urlencoded" },
		body: new URLSearchParams({ grant_type: "refresh_token", refresh_token: current.refreshToken }),
	});
	if (!res.ok) {
		throw new Error(`token refresh failed: ${res.status}`);
	}
	const body = await res.json();
	const next: Session = {
		accessToken: body.access_token,
		refreshToken: body.refresh_token ?? current.refreshToken,
		expiresAt: Date.now() + body.expires_in * 1000,
	};
	await store.save(next);
	return next;
}
