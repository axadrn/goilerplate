<script lang="ts">
	import Clock3Icon from '@lucide/svelte/icons/clock-3';
	import DollarSignIcon from '@lucide/svelte/icons/dollar-sign';
	import TargetIcon from '@lucide/svelte/icons/target';
	import TrendingUpIcon from '@lucide/svelte/icons/trending-up';
	import UsersRoundIcon from '@lucide/svelte/icons/users-round';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import AccountsChart from './accounts-chart.svelte';
	import RevenueChart from './revenue-chart.svelte';

	const stats = [
		{ label: 'Monthly revenue', value: '$48,240', detail: '+12.5% from last month', icon: DollarSignIcon },
		{ label: 'Active accounts', value: '2,420', detail: '+180 this month', icon: UsersRoundIcon },
		{ label: 'Trial conversion', value: '18.2%', detail: '+2.4 points this quarter', icon: TargetIcon },
		{ label: 'Support response', value: '1h 18m', detail: 'Within the 2 hour target', icon: Clock3Icon }
	];
</script>

<div class="space-y-6 p-4 sm:p-6">
	<div class="flex items-end justify-between gap-4">
		<div>
			<h1 class="text-3xl font-bold tracking-tight">Dashboard</h1>
			<p class="mt-2 text-sm text-muted-foreground">A clear starting point for your product's most important signals.</p>
		</div>
		<Badge variant="outline" class="hidden font-normal text-muted-foreground sm:inline-flex">
			<span class="size-2 rounded-full bg-chart-2"></span>
			Example data
		</Badge>
	</div>
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
		{#each stats as stat (stat.label)}
			<Card.Root class="gap-0">
				<Card.Header>
					<Card.Description>{stat.label}</Card.Description>
					<Card.Title class="text-2xl font-semibold tabular-nums sm:text-3xl">{stat.value}</Card.Title>
					<Card.Action>
						<div class="flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">
							<stat.icon class="size-4" />
						</div>
					</Card.Action>
				</Card.Header>
				<Card.Content class="mt-4">
					<p class="flex items-center gap-1.5 text-xs text-muted-foreground">
						<span class="size-1 rounded-full bg-chart-2"></span>
						{stat.detail}
					</p>
				</Card.Content>
			</Card.Root>
		{/each}
	</div>
	<div class="grid gap-4 xl:grid-cols-2">
		<Card.Root class="gap-0">
			<Card.Header class="border-b pb-4">
				<Card.Title>Revenue trend</Card.Title>
				<Card.Description>Monthly recurring revenue over twelve months.</Card.Description>
				<Card.Action>
					<Badge variant="outline" class="font-normal text-muted-foreground">
						<TrendingUpIcon class="size-3" />
						12.5%
					</Badge>
				</Card.Action>
			</Card.Header>
			<Card.Content class="pt-4">
				<RevenueChart />
			</Card.Content>
		</Card.Root>
		<Card.Root class="gap-0">
			<Card.Header class="border-b pb-4">
				<Card.Title>Account growth</Card.Title>
				<Card.Description>Trials and paid upgrades for the last six months.</Card.Description>
				<Card.Action>
					<Badge variant="outline" class="font-normal text-muted-foreground">
						<UsersRoundIcon class="size-3" />
						+38 upgrades
					</Badge>
				</Card.Action>
			</Card.Header>
			<Card.Content class="pt-4">
				<AccountsChart />
			</Card.Content>
		</Card.Root>
	</div>
</div>
