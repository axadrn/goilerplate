<script lang="ts">
	import { on } from 'svelte/events';
	import { Toaster } from 'svelte-sonner';

	// static/theme.js sets the dark class and announces each change.
	const currentTheme = () => (document.documentElement.classList.contains('dark') ? 'dark' : 'light');
	let theme = $state<'light' | 'dark'>(currentTheme());

	$effect(() => on(document, 'theme-changed', () => (theme = currentTheme())));
</script>

<Toaster
	{theme}
	class="toaster group"
	style="--normal-bg: var(--color-popover); --normal-text: var(--color-popover-foreground); --normal-border: var(--color-border);"
/>
