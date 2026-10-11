import type { Action } from "svelte/action";

/** Focuses the element once it is mounted. */
export const autofocus: Action<HTMLElement> = (node) => {
	queueMicrotask(() => node.focus());
};
