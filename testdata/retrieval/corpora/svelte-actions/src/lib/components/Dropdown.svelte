<script lang="ts">
	import type { Snippet } from "svelte";
	import { clickOutside } from "$lib/actions/clickOutside";

	interface Props {
		label: string;
		children: Snippet;
	}

	let { label, children }: Props = $props();

	let open = $state(false);

	function toggle() {
		open = !open;
	}

	function close() {
		open = false;
	}

	$effect(() => {
		if (!open) return;
		const onKeydown = (event: KeyboardEvent) => {
			if (event.key === "Escape") close();
		};
		window.addEventListener("keydown", onKeydown);
		return () => window.removeEventListener("keydown", onKeydown);
	});
</script>

<div class="dropdown" use:clickOutside={close}>
	<button type="button" aria-expanded={open} onclick={toggle}>{label}</button>
	{#if open}
		<div class="dropdown-menu" role="menu">
			{@render children()}
		</div>
	{/if}
</div>

<style>
	.dropdown {
		position: relative;
	}

	.dropdown-menu {
		position: absolute;
		top: 100%;
		min-width: 12rem;
	}
</style>
