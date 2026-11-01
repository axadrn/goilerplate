<script lang="ts">
	import { page } from '$app/state';
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import CircleAlertIcon from '@lucide/svelte/icons/circle-alert';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import { config } from '#lib/config.js';

	const notFound = $derived(page.status === 404);
	const title = $derived(page.error?.title ?? (notFound ? 'Page not found' : 'Something went wrong'));
</script>

<svelte:head>
	<title>{title} - {config.name}</title>
</svelte:head>

<Empty.Root class="min-h-svh rounded-none border-0">
	<Empty.Header>
		{#if notFound}
			<Empty.Media>
				<span class="text-8xl font-bold tracking-tighter text-muted-foreground/25">404</span>
			</Empty.Media>
		{:else}
			<Empty.Media variant="icon">
				<CircleAlertIcon class="size-5" />
			</Empty.Media>
			<p class="text-sm font-medium text-muted-foreground">{page.status}</p>
		{/if}
		<Empty.Title class="text-xl">{title}</Empty.Title>
		<Empty.Description>
			{notFound ? 'The page you are looking for does not exist or has moved.' : page.error?.message}
		</Empty.Description>
	</Empty.Header>
	<Empty.Content>
		<Button href="/">
			<ArrowLeftIcon class="size-4" />
			Back to home
		</Button>
	</Empty.Content>
</Empty.Root>
