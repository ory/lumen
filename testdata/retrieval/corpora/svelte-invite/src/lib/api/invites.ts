export interface Invite {
	token: string;
	email: string;
	teamName: string;
	expiresAt: string;
}

export async function fetchInvite(token: string): Promise<Invite | null> {
	const res = await fetch(`/api/invites/${encodeURIComponent(token)}`);
	if (res.status === 404) return null;
	if (!res.ok) throw new Error(`invite lookup failed: ${res.status}`);
	return (await res.json()) as Invite;
}

export async function acceptInvite(token: string): Promise<void> {
	const res = await fetch(`/api/invites/${encodeURIComponent(token)}/accept`, { method: "POST" });
	if (!res.ok) throw new Error(`accept failed: ${res.status}`);
}

export async function sendInvite(email: string, teamId: string): Promise<void> {
	const res = await fetch(`/api/teams/${teamId}/invites`, {
		method: "POST",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({ email }),
	});
	if (!res.ok) throw new Error(`send failed: ${res.status}`);
}
