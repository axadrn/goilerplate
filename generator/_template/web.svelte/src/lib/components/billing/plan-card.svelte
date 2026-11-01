<script lang="ts">
	import CheckIcon from '@lucide/svelte/icons/check';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import type { BillingInterval, Plan } from '#lib/types.js';
	import { cn } from '#lib/utils.js';

	let {
		plan,
		interval,
		purchasable,
		pending,
		oncheckout
	}: {
		plan: Plan;
		interval: BillingInterval;
		// False once the provider has a subscription; it is changed in the
		// provider portal then.
		purchasable: boolean;
		pending: boolean;
		oncheckout: (plan: Plan) => void;
	} = $props();

	// The catalog's featured plan, like the HTML page.
	const featured = $derived(plan.id === 'pro');
	const price = $derived(plan.prices.find((candidate) => candidate.interval === interval));

	function submit(event: SubmitEvent) {
		event.preventDefault();
		oncheckout(plan);
	}
</script>

<Card.Root class={cn('relative flex h-full flex-col overflow-visible', featured && 'border-primary ring-2 ring-primary')}>
	{#if featured}
		<div class="absolute -top-4 left-1/2 -translate-x-1/2">
			<Badge>Most Popular</Badge>
		</div>
	{/if}
	<Card.Header>
		<Card.Title class="text-2xl">{plan.name}</Card.Title>
		{#if price}
			<div class="mt-2 flex items-baseline gap-1">
				<span class="text-4xl font-bold">{price.display_price}</span>
				<span class="text-muted-foreground">/ {interval === 'yearly' ? 'year' : 'month'}</span>
			</div>
		{/if}
		<Card.Description class="mt-2">{plan.description}</Card.Description>
	</Card.Header>
	<Card.Content class="flex-grow">
		<ul class="space-y-3">
			{#each plan.benefits as benefit (benefit)}
				<li class="flex items-start gap-2">
					<CheckIcon class="mt-0.5 size-5 shrink-0 text-primary" />
					<span class="text-sm">{benefit}</span>
				</li>
			{/each}
		</ul>
	</Card.Content>
	{#if plan.current || purchasable}
		<Card.Footer class="mt-auto">
			{#if plan.current}
				<Button variant="outline" class="w-full" disabled>Current Plan</Button>
			{:else}
				<form class="w-full" onsubmit={submit} aria-busy={pending || undefined} inert={pending}>
					<Button type="submit" class="w-full" variant={featured ? 'default' : 'outline'}>
						Upgrade to {plan.name}
					</Button>
				</form>
			{/if}
		</Card.Footer>
	{/if}
</Card.Root>
