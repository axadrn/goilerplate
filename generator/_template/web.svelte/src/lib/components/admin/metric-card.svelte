<script lang="ts">
	import type UsersIcon from '@lucide/svelte/icons/users';
	import * as Card from '#lib/components/ui/card/index.js';

	let { label, value, Icon }: { label: string; value: number; Icon: typeof UsersIcon } = $props();
</script>

<Card.Root>
	<Card.Content class="flex items-center justify-between">
		<div>
			<p class="text-sm text-muted-foreground">{label}</p>
			<p class="mt-1 text-2xl font-semibold">{value}</p>
		</div>
		<span class="rounded-lg bg-muted p-2 text-muted-foreground">
			<Icon class="size-5" />
		</span>
	</Card.Content>
</Card.Root>
