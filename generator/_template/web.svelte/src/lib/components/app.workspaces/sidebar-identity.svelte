<script lang="ts">
	import { page } from '$app/state';
	import Building2Icon from '@lucide/svelte/icons/building-2';
	import CheckIcon from '@lucide/svelte/icons/check';
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { session } from '#lib/session.svelte.js';
	import { roleLabels } from '#lib/workspace/workspace.js';

	const current = $derived(page.data.workspace?.workspace);
	const memberships = $derived(session.account?.workspaces ?? []);
</script>

{#if current}
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Sidebar.MenuButton size="lg" {...props}>
					<span
						class="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sm font-semibold text-sidebar-primary-foreground"
					>
						{current.name.charAt(0).toUpperCase()}
					</span>
					<div class="min-w-0 flex-1 text-left">
						<span class="block truncate text-sm font-bold">{current.name}</span>
						<span class="block text-xs text-muted-foreground">Switch workspace</span>
					</div>
					<ChevronsUpDownIcon class="ml-auto size-4" />
				</Sidebar.MenuButton>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content class="w-72" side="bottom" align="start">
			<DropdownMenu.Label>Workspaces</DropdownMenu.Label>
			{#each memberships as membership (membership.workspace.id)}
				<DropdownMenu.Item class="py-2">
					{#snippet child({ props })}
						<a href="/app/{membership.workspace.slug}/dashboard" {...props}>
							<Building2Icon class="size-4 shrink-0" />
							<div class="min-w-0 flex-1">
								<span class="block truncate">{membership.workspace.name}</span>
								<span class="block text-xs text-muted-foreground">{roleLabels[membership.role]}</span>
							</div>
							{#if membership.workspace.id === current.id}
								<CheckIcon class="ml-auto size-4 shrink-0" />
								<span class="sr-only">Current workspace</span>
							{/if}
						</a>
					{/snippet}
				</DropdownMenu.Item>
			{/each}
			<DropdownMenu.Separator />
			<DropdownMenu.Item>
				{#snippet child({ props })}
					<a href="/app/{current.slug}/workspace" {...props}>
						<SettingsIcon class="size-4" />
						Workspace settings
					</a>
				{/snippet}
			</DropdownMenu.Item>
			<DropdownMenu.Item>
				{#snippet child({ props })}
					<a href="/workspaces/new" {...props}>
						<PlusIcon class="size-4" />
						New workspace
					</a>
				{/snippet}
			</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
{/if}
