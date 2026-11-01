<script lang="ts">
	import { curveNatural } from 'd3-shape';
	import { AreaChart } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';

	const data = [
		{ month: 'Jan', revenue: 24800 },
		{ month: 'Feb', revenue: 27100 },
		{ month: 'Mar', revenue: 29600 },
		{ month: 'Apr', revenue: 31400 },
		{ month: 'May', revenue: 33800 },
		{ month: 'Jun', revenue: 35700 },
		{ month: 'Jul', revenue: 38200 },
		{ month: 'Aug', revenue: 40100 },
		{ month: 'Sep', revenue: 42300 },
		{ month: 'Oct', revenue: 44700 },
		{ month: 'Nov', revenue: 46100 },
		{ month: 'Dec', revenue: 48240 }
	];

	const config = {
		revenue: { label: 'Revenue', color: 'var(--chart-1)' }
	} satisfies Chart.ChartConfig;
</script>

<Chart.Container {config} class="aspect-auto h-[320px] min-h-[320px] w-full">
	<AreaChart
		{data}
		x="month"
		axis="x"
		series={[{ key: 'revenue', label: config.revenue.label, color: 'var(--color-revenue)' }]}
		props={{
			area: { curve: curveNatural, fillOpacity: 0.2, line: { class: 'stroke-1' } },
			xAxis: { tickMarks: false }
		}}
	>
		{#snippet tooltip()}
			<Chart.Tooltip indicator="line" />
		{/snippet}
	</AreaChart>
</Chart.Container>
