<script lang="ts">
	import MessageCircleQuestionMarkIcon from '@lucide/svelte/icons/message-circle-question-mark';
	import type { Snippet } from 'svelte';
	import * as Sidebar from '#lib/components/ui/sidebar/index.js';
	import type { Account } from '#lib/types.js';
	import AppHeader from './app-header.svelte';
	// sidebar-identity and sidebar-navigation differ with workspaces.
	import SidebarIdentity from './sidebar-identity.svelte';
	import SidebarNavigation from './sidebar-navigation.svelte';
	import UserMenu from './user-menu.svelte';

	let {
		account,
		supportEmail,
		title,
		children
	}: { account: Account; supportEmail?: string; title: string; children: Snippet } = $props();
</script>

<Sidebar.Provider
	style="--sidebar-width: calc(var(--spacing) * 72); --sidebar-width-icon: 3rem; --header-height: calc(var(--spacing) * 12);"
>
	<Sidebar.Root collapsible="offcanvas" variant="inset">
		<Sidebar.Header>
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<SidebarIdentity />
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Header>
		<Sidebar.Content>
			<Sidebar.Group>
				<Sidebar.GroupLabel>Overview</Sidebar.GroupLabel>
				<Sidebar.Menu>
					<SidebarNavigation />
				</Sidebar.Menu>
			</Sidebar.Group>
		</Sidebar.Content>
		<Sidebar.Footer>
			<Sidebar.Menu>
				{#if supportEmail}
					<Sidebar.MenuItem>
						<Sidebar.MenuButton size="sm">
							{#snippet child({ props })}
								<a href="mailto:{supportEmail}" {...props}>
									<MessageCircleQuestionMarkIcon class="size-4" />
									<span>Support</span>
								</a>
							{/snippet}
						</Sidebar.MenuButton>
					</Sidebar.MenuItem>
				{/if}
			</Sidebar.Menu>
			<Sidebar.Separator />
			<Sidebar.Menu>
				<Sidebar.MenuItem>
					<UserMenu {account} />
				</Sidebar.MenuItem>
			</Sidebar.Menu>
		</Sidebar.Footer>
		<Sidebar.Rail />
	</Sidebar.Root>
	<Sidebar.Inset>
		<AppHeader {title} />
		<!-- Sidebar.Inset is the page's main element. -->
		<div class="flex-1">
			{@render children()}
		</div>
	</Sidebar.Inset>
</Sidebar.Provider>
