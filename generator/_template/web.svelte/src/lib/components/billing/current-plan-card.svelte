<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatSubscriptionPrice } from '#lib/billing/billing.js';
	import { formatLongDay } from '#lib/format.js';
	import type { Access, Subscription } from '#lib/types.js';
	import SubscriptionBadge from './subscription-badge.svelte';

	let {
		subscription,
		access,
		manageLabel,
		manageable,
		pending,
		onmanage
	}: {
		subscription: Subscription | null;
		access: Access | null;
		manageLabel: string;
		// Whether the payment provider knows a customer to manage.
		manageable: boolean;
		pending: boolean;
		onmanage: () => void;
	} = $props();

	const planId = $derived(subscription?.plan_id ?? access?.plan_id ?? 'free');
	const price = $derived(subscription && planId !== 'free' ? formatSubscriptionPrice(subscription) : '');
	const periodEnd = $derived(planId !== 'free' ? subscription?.current_period_end : null);
	const renews = $derived(
		subscription?.status === 'active' || subscription?.status === 'trialing' || subscription?.status === 'past_due'
	);
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Current Plan</Card.Title>
		<Card.Description>Your active subscription</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="flex flex-wrap items-center justify-between gap-4">
			<div>
				<div class="mb-2 flex flex-wrap items-center gap-3">
					<h3 class="text-2xl font-bold capitalize">{planId}</h3>
					{#if price}
						<span class="text-lg text-muted-foreground">{price}</span>
					{/if}
					{#if subscription}
						<SubscriptionBadge status={subscription.status} />
					{/if}
				</div>
				{#if periodEnd && renews}
					<p class="text-sm text-muted-foreground">Next billing date: {formatLongDay(periodEnd)}</p>
				{:else if periodEnd && subscription?.status === 'cancelled'}
					<p class="text-sm text-muted-foreground">Access until: {formatLongDay(periodEnd)}</p>
				{/if}
				{#if access?.state === 'grace' && access.access_ends_at && subscription?.status !== 'cancelled'}
					<p class="text-sm text-muted-foreground">Access until: {formatLongDay(access.access_ends_at)}</p>
				{/if}
			</div>
			{#if manageable}
				<Button variant="outline" aria-busy={pending || undefined} onclick={onmanage}>{manageLabel}</Button>
			{/if}
		</div>
	</Card.Content>
</Card.Root>
