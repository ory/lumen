import type { Action } from "svelte/action";

/**
 * Calls `callback` when a pointer press lands outside `node`.
 */
export const clickOutside: Action<HTMLElement, () => void> = (node, callback) => {
	let handler = callback;

	function onPointerDown(event: PointerEvent) {
		if (!node.contains(event.target as Node)) {
			handler?.();
		}
	}

	document.addEventListener("pointerdown", onPointerDown, true);

	return {
		update(next) {
			handler = next;
		},
		destroy() {
			document.removeEventListener("pointerdown", onPointerDown, true);
		},
	};
};
