<script lang="ts">
	import { page } from '$app/state';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import Building2Icon from '@lucide/svelte/icons/building-2';
	import HouseIcon from '@lucide/svelte/icons/house';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';

	const workspace = $derived(page.data.workspace);
	const basePath = $derived(`/app/${workspace?.workspace.slug}`);
</script>

{#snippet link(href: string, label: string, Icon: typeof HouseIcon)}
	<Sidebar.MenuItem>
		<Sidebar.MenuButton isActive={page.url.pathname === href}>
			{#snippet child({ props })}
				<a {href} {...props}>
					<Icon class="size-4" />
					<span>{label}</span>
				</a>
			{/snippet}
		</Sidebar.MenuButton>
	</Sidebar.MenuItem>
{/snippet}

{#if workspace}
	{@render link(`${basePath}/dashboard`, 'Dashboard', HouseIcon)}
	{@render link(`${basePath}/workspace`, 'Workspace', Building2Icon)}
	{#if workspace.permissions.view_audit_log}
		{@render link(`${basePath}/workspace/activity`, 'Activity', ActivityIcon)}
	{/if}
{/if}
