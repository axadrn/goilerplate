<script lang="ts">
	import { page } from '$app/state';
	import HouseIcon from '@lucide/svelte/icons/house';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
</script>

<Sidebar.MenuItem>
	<Sidebar.MenuButton isActive={page.url.pathname === '/app/dashboard'}>
		{#snippet child({ props })}
			<a href="/app/dashboard" {...props}>
				<HouseIcon class="size-4" />
				<span>Dashboard</span>
			</a>
		{/snippet}
	</Sidebar.MenuButton>
</Sidebar.MenuItem>
