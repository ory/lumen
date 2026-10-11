<script lang="ts">
	import { goto } from "$app/navigation";
	import { fetchInvite, acceptInvite, type Invite } from "$lib/api/invites";

	interface Props {
		token: string;
	}

	let { token }: Props = $props();

	let invite: Invite | null = $state(null);
	let status: "loading" | "valid" | "expired" | "missing" = $state("loading");
	let accepting = $state(false);

	function validateInviteToken(value: string): boolean {
		return /^[A-Za-z0-9_-]{32}$/.test(value);
	}

	$effect(() => {
		if (!validateInviteToken(token)) {
			status = "missing";
			return;
		}
		fetchInvite(token).then((result) => {
			invite = result;
			if (!result) status = "missing";
			else status = Date.parse(result.expiresAt) < Date.now() ? "expired" : "valid";
		});
	});

	async function accept() {
		if (!invite) return;
		accepting = true;
		try {
			await acceptInvite(invite.token);
			await goto(`/teams/${invite.teamName}`);
		} finally {
			accepting = false;
		}
	}
</script>

<section class="invite-card">
	{#if status === "loading"}
		<p class="invite-hint">Checking your invitation…</p>
	{:else if status === "expired"}
		<p class="invite-error" role="alert">Invite expired. Ask a team admin to send a new link.</p>
	{:else if status === "missing"}
		<p class="invite-error" role="alert">This invitation link is not valid.</p>
	{:else if invite}
		<h2>Join {invite.teamName}</h2>
		<p class="invite-hint">Signed in as {invite.email}</p>
		<button onclick={accept} disabled={accepting}>Accept invitation</button>
	{/if}
</section>

<style>
	.invite-card {
		max-width: 28rem;
		margin: 4rem auto;
		padding: 2rem;
		border-radius: 0.75rem;
		box-shadow: 0 1px 3px rgb(0 0 0 / 0.12);
	}

	.invite-error {
		color: #b42318;
		background: #fef3f2;
		border-left: 3px solid #f04438;
		padding: 0.75rem 1rem;
	}

	.invite-hint {
		color: #475467;
		font-size: 0.875rem;
	}
</style>
