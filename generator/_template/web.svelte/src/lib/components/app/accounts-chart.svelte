<script lang="ts">
	import { BarChart } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';

	const data = [
		{ month: 'Jul', trials: 210, upgrades: 52 },
		{ month: 'Aug', trials: 248, upgrades: 61 },
		{ month: 'Sep', trials: 232, upgrades: 58 },
		{ month: 'Oct', trials: 276, upgrades: 72 },
		{ month: 'Nov', trials: 301, upgrades: 79 },
		{ month: 'Dec', trials: 324, upgrades: 88 }
	];

	const config = {
		trials: { label: 'Trials', color: 'var(--chart-2)' },
		upgrades: { label: 'Upgrades', color: 'var(--chart-1)' }
	} satisfies Chart.ChartConfig;
</script>

<Chart.Container {config} class="aspect-auto h-[320px] min-h-[320px] w-full">
	<BarChart
		{data}
		x="month"
		axis="x"
		seriesLayout="group"
		legend
		rule={false}
		series={[
			{ key: 'trials', label: config.trials.label, color: 'var(--color-trials)' },
			{ key: 'upgrades', label: config.upgrades.label, color: 'var(--color-upgrades)' }
		]}
		props={{
			bars: { stroke: 'none', strokeWidth: 0, rounded: 'top', radius: 4 },
			highlight: { area: { fill: 'none' } }
		}}
	>
		{#snippet tooltip()}
			<Chart.Tooltip indicator="dot" />
		{/snippet}
	</BarChart>
</Chart.Container>
