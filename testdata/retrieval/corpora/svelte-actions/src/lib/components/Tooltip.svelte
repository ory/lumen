<script lang="ts">
	import type { Snippet } from "svelte";

	interface Props {
		text: string;
		children: Snippet;
	}

	let { text, children }: Props = $props();

	let visible = $state(false);

	function show() {
		visible = true;
	}

	function hide() {
		visible = false;
	}
</script>

<span class="tooltip" onpointerenter={show} onpointerleave={hide} role="presentation">
	{@render children()}
	{#if visible}
		<span class="tooltip-text" role="tooltip">{text}</span>
	{/if}
</span>
