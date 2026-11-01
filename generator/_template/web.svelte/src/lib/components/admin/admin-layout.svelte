<script lang="ts">
	import ArrowLeftIcon from '@lucide/svelte/icons/arrow-left';
	import ShieldIcon from '@lucide/svelte/icons/shield';
	import type { Snippet } from 'svelte';
	import ThemeSwitcher from '#lib/components/app/theme-switcher.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { config } from '#lib/config.js';

	let { children }: { children: Snippet } = $props();
</script>

<div class="min-h-screen bg-muted/20">
	<header class="border-b bg-background">
		<div class="flex h-16 w-full items-center gap-4 px-6 lg:px-8">
			<a href="/admin" class="flex min-w-0 items-center gap-2 font-semibold">
				<span class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
					<ShieldIcon class="size-4" />
				</span>
				<span class="truncate">{config.name} Admin</span>
			</a>
			<div class="ml-auto flex shrink-0 items-center gap-2">
				<ThemeSwitcher />
				<Button href="/app" variant="outline" size="sm">
					<ArrowLeftIcon class="size-4" />
					Back to app
				</Button>
			</div>
		</div>
	</header>
	<main class="w-full px-6 py-8 lg:px-8">
		{@render children()}
	</main>
</div>
