<script lang="ts">
	import GalleryVerticalEndIcon from '@lucide/svelte/icons/gallery-vertical-end';
	import type { Snippet } from 'svelte';
	import ThemeSwitcher from '#lib/components/app/theme-switcher.svelte';
	import { config } from '#lib/config.js';

	let { children }: { children: Snippet } = $props();
</script>

<div class="relative">
	<div class="absolute top-4 right-4 z-10">
		<ThemeSwitcher />
	</div>
	<main class="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6 md:p-10">
		<div class="flex w-full max-w-sm flex-col gap-6">
			<a href="/" class="flex items-center gap-2 self-center font-medium">
				<div class="flex size-6 items-center justify-center rounded-md bg-primary text-primary-foreground">
					<GalleryVerticalEndIcon class="size-4" />
				</div>
				{config.name}
			</a>
			{@render children()}
		</div>
	</main>
</div>
