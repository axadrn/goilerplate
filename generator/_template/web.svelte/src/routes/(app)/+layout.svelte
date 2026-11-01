<script lang="ts">
	import { page } from '$app/state';
	import AppShell from '#lib/components/app/app-shell.svelte';
	import { config } from '#lib/config.js';
	import { session } from '#lib/session.svelte.js';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<title>{page.data.title ? `${page.data.title} - ${config.name}` : config.name}</title>
</svelte:head>

{#if session.account}
	<AppShell
		account={session.account}
		supportEmail={session.meta?.support_email}
		title={page.data.title ?? ''}
	>
		{@render children()}
	</AppShell>
{/if}
