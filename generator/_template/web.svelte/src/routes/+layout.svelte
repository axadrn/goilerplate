<script lang="ts">
	import '../app.css';
	import { invalidateAll } from '$app/navigation';
	import Toaster from '#lib/components/app/toaster.svelte';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// Back or Forward can restore a page from the browser cache, for example
	// the app after a sign-out. Load it again so the guards ask the server.
	function pageshow(event: PageTransitionEvent) {
		if (event.persisted) {
			invalidateAll();
		}
	}
</script>

<svelte:window onpageshow={pageshow} />

{@render children()}
<Toaster />
