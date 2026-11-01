<script lang="ts">
	import { Badge } from '#lib/components/ui/badge/index.js';
	import type { SubscriptionStatus } from '#lib/types.js';

	let { status }: { status: SubscriptionStatus } = $props();
</script>

{#if status === 'active'}
	<Badge class="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">Active</Badge>
{:else if status === 'trialing'}
	<Badge class="bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">Trial</Badge>
{:else if status === 'past_due'}
	<Badge class="bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200">Past due</Badge>
{:else if status === 'incomplete'}
	<Badge variant="outline">Payment pending</Badge>
{:else if status === 'inactive'}
	<Badge variant="destructive">Access ended</Badge>
{:else if status === 'cancelled'}
	<Badge variant="destructive">Cancelled</Badge>
{:else}
	<Badge variant="secondary">{status}</Badge>
{/if}
