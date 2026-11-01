<script lang="ts">
	import ChevronsUpDownIcon from '@lucide/svelte/icons/chevrons-up-down';
	import LogOutIcon from '@lucide/svelte/icons/log-out';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import { logout } from '#lib/logout.js';
	import type { Account } from '#lib/types.js';
	// account-links differs with workspaces.
	import AccountLinks from './account-links.svelte';
	import UserAvatar from './user-avatar.svelte';

	let { account }: { account: Account } = $props();
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Sidebar.MenuButton size="lg" {...props}>
				<UserAvatar {account} />
				<div class="grid flex-1 text-left text-sm leading-tight">
					<span class="truncate font-medium">{account.user.name}</span>
					<span class="truncate text-xs text-muted-foreground">{account.user.email}</span>
				</div>
				<ChevronsUpDownIcon class="ml-auto size-4" />
			</Sidebar.MenuButton>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content class="w-56" side="top" align="start">
		<DropdownMenu.Label>
			<div class="flex flex-col">
				<span class="font-medium">{account.user.name}</span>
				<span class="text-xs text-muted-foreground">{account.user.email}</span>
			</div>
		</DropdownMenu.Label>
		<DropdownMenu.Separator />
		<AccountLinks />
		<DropdownMenu.Separator />
		<DropdownMenu.Item onSelect={logout}>
			<span class="flex items-center">
				<LogOutIcon class="mr-2 size-4" />
				Log out
			</span>
		</DropdownMenu.Item>
	</DropdownMenu.Content>
</DropdownMenu.Root>
