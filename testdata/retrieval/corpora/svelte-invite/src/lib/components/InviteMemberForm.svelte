<script lang="ts">
	import { sendInvite } from "$lib/api/invites";

	interface Props {
		teamId: string;
	}

	let { teamId }: Props = $props();

	let email = $state("");
	let error = $state("");
	let sent = $state(false);

	function validateEmail(value: string): boolean {
		return /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(value);
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = "";
		if (!validateEmail(email)) {
			error = "Enter a valid email address.";
			return;
		}
		try {
			await sendInvite(email, teamId);
			sent = true;
			email = "";
		} catch {
			error = "Could not send the invitation.";
		}
	}
</script>

<form onsubmit={submit} class="invite-form">
	<label>
		Email
		<input type="email" bind:value={email} placeholder="teammate@example.com" />
	</label>
	{#if error}
		<p class="form-error">{error}</p>
	{/if}
	{#if sent}
		<p class="form-success">Invitation sent.</p>
	{/if}
	<button type="submit">Send invite</button>
</form>

<style>
	.invite-form {
		display: grid;
		gap: 0.5rem;
	}

	.form-error {
		color: #b42318;
		font-size: 0.875rem;
	}

	.form-success {
		color: #027a48;
	}
</style>
