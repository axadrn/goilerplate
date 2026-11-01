<script lang="ts">
	import * as Avatar from '#lib/components/ui/avatar/index.js';
	import type { Account } from '#lib/types.js';

	let { account }: { account: Account } = $props();

	const initial = $derived(account.user.name.charAt(0).toUpperCase());
</script>

<Avatar.Root class="size-8 rounded-lg after:rounded-lg">
	{#if account.avatar_url}
		<Avatar.Image src={account.avatar_url} alt={account.user.name} class="rounded-lg" />
	{/if}
	<Avatar.Fallback class="rounded-lg bg-muted text-xs font-semibold text-foreground">{initial}</Avatar.Fallback>
</Avatar.Root>
