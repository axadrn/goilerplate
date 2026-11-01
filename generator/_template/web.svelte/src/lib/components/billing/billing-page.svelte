<script lang="ts">
	import { tick } from 'svelte';
	import * as Tabs from '#lib/components/ui/tabs/index.js';
	import {
		billingPaths,
		hasBillingAccount,
		manageURL,
		notifyCheckoutFailure,
		notifyManageFailure,
		startCheckout
	} from '#lib/billing/billing.js';
	import type { Billing, BillingInterval, Plan } from '#lib/types.js';
	import CurrentPlanCard from './current-plan-card.svelte';
	import PlanCard from './plan-card.svelte';

	let {
		billing,
		description,
		workspaceSlug
	}: {
		billing: Billing;
		description: string;
		// The workspace whose billing this is, with workspaces.
		workspaceSlug?: string;
	} = $props();

	const paths = $derived(billingPaths(workspaceSlug));
	const manageable = $derived(hasBillingAccount(billing.subscription));

	let interval = $state<BillingInterval>('monthly');
	// The plan whose checkout is starting, or "manage" while the portal opens.
	let pending = $state<string | null>(null);

	async function checkout(plan: Plan) {
		if (pending) {
			return;
		}
		pending = plan.id;
		try {
			window.location.assign(await startCheckout(paths, plan.id, interval));
		} catch (cause) {
			pending = null;
			notifyCheckoutFailure(cause);
			await tick();
			document.querySelector<HTMLElement>(`[data-plan="${plan.id}-${interval}"] button[type="submit"]`)?.focus();
		}
	}

	async function manage() {
		if (pending) {
			return;
		}
		pending = 'manage';
		try {
			window.location.assign(await manageURL(paths));
		} catch (cause) {
			pending = null;
			notifyManageFailure(
				cause,
				workspaceSlug
					? 'Billing management could not be opened. Please try again.'
					: 'The customer portal could not be opened. Please try again.'
			);
		}
	}

	// Back from the provider can restore this page from the cache with a
	// request still marked as pending.
	function pageshow(event: PageTransitionEvent) {
		if (event.persisted) {
			pending = null;
		}
	}
</script>

<svelte:window onpageshow={pageshow} />

{#snippet plans(current: BillingInterval)}
	<div class={['grid gap-4', billing.plans.length === 2 ? 'md:grid-cols-2' : 'md:grid-cols-3']}>
		{#each billing.plans as plan (plan.id)}
			<div class="h-full" data-plan="{plan.id}-{current}">
				<PlanCard
					{plan}
					interval={current}
					purchasable={!manageable}
					pending={pending === plan.id && interval === current}
					oncheckout={checkout}
				/>
			</div>
		{/each}
	</div>
{/snippet}

<div class="container max-w-6xl px-6 py-8">
	<div class="mb-8">
		<h1 class="text-3xl font-bold">Billing</h1>
		<p class="mt-2 text-muted-foreground">{description}</p>
	</div>
	<div class="space-y-6">
		<CurrentPlanCard
			subscription={billing.subscription}
			access={billing.access}
			manageLabel={workspaceSlug ? 'Manage subscription' : 'Manage Subscription'}
			{manageable}
			pending={pending === 'manage'}
			onmanage={manage}
		/>
		<Tabs.Root bind:value={() => interval, (value) => (interval = value as BillingInterval)} class="gap-0">
			<div class="mb-4 flex flex-wrap items-center justify-between gap-4">
				<h3 class="text-xl font-bold">Available Plans</h3>
				<Tabs.List aria-label="Billing interval">
					<Tabs.Trigger value="monthly">Monthly</Tabs.Trigger>
					<Tabs.Trigger value="yearly">
						Yearly
						<span class="ml-1 text-xs font-semibold text-green-600 dark:text-green-400">Save 20%</span>
					</Tabs.Trigger>
				</Tabs.List>
			</div>
			<Tabs.Content value="monthly">
				{@render plans('monthly')}
			</Tabs.Content>
			<Tabs.Content value="yearly">
				{@render plans('yearly')}
			</Tabs.Content>
		</Tabs.Root>
	</div>
</div>
