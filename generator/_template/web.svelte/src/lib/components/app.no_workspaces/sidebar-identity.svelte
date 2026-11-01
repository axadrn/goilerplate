<script lang="ts">
	import LayoutDashboardIcon from '@lucide/svelte/icons/layout-dashboard';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { config } from '#lib/config.js';
</script>

<Sidebar.MenuButton size="lg">
	{#snippet child({ props })}
		<a href="/app/dashboard" {...props}>
			<LayoutDashboardIcon />
			<div class="flex flex-col">
				<span class="text-sm font-bold">{config.name}</span>
				<span class="text-xs text-muted-foreground">Application</span>
			</div>
		</a>
	{/snippet}
</Sidebar.MenuButton>
