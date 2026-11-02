<script lang="ts">
	import { page } from '$app/state';
	import AuthLayout from '#lib/components/auth/auth-layout.svelte';
	import { config } from '#lib/config.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	{#if page.data.title}
		<title>{page.data.title} · {config.name}</title>
	{/if}
</svelte:head>

<AuthLayout>
	{@render children()}
</AuthLayout>
