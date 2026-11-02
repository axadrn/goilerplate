<script lang="ts">
	import { page } from '$app/state';
	import AdminLayout from '#lib/components/admin/admin-layout.svelte';
	import { config } from '#lib/config.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<title>{page.data.title ? `${page.data.title} · ${config.name}` : config.name}</title>
</svelte:head>

<AdminLayout>
	{@render children()}
</AdminLayout>
