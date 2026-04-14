<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface DashboardProps {
		userId: string;
		refreshInterval?: number;
	}

	let { userId, refreshInterval = 30000 }: DashboardProps = $props();

	interface ActivityEntry {
		id: string;
		action: string;
		timestamp: number;
	}

	class ActivityCache {
		private entries: ActivityEntry[] = [];

		add(entry: ActivityEntry): void {
			this.entries = [entry, ...this.entries].slice(0, 50);
		}

		clear(): void {
			this.entries = [];
		}

		recent(): ActivityEntry[] {
			return this.entries;
		}
	}

	const cache = new ActivityCache();
	let activity = $state<ActivityEntry[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let intervalId: ReturnType<typeof setInterval> | null = null;

	async function loadUserActivity(): Promise<void> {
		loading = true;
		error = null;
		try {
			const resp = await fetch(`/api/users/${userId}/activity`);
			if (!resp.ok) throw new Error(`status ${resp.status}`);
			const data: ActivityEntry[] = await resp.json();
			data.forEach(e => cache.add(e));
			activity = cache.recent();
		} catch (e) {
			error = e instanceof Error ? e.message : 'unknown error';
		} finally {
			loading = false;
		}
	}

	function handleRefresh(): void {
		void loadUserActivity();
	}

	onMount(() => {
		void loadUserActivity();
		intervalId = setInterval(handleRefresh, refreshInterval);
	});

	onDestroy(() => {
		if (intervalId !== null) clearInterval(intervalId);
		cache.clear();
	});
</script>

<div class="dashboard">
	{#if loading}<p>Loading…</p>{/if}
	{#if error}<p class="error">{error}</p>{/if}
	{#each activity as entry (entry.id)}
		<div class="entry">{entry.action}</div>
	{/each}
	<button onclick={handleRefresh}>Refresh</button>
</div>
