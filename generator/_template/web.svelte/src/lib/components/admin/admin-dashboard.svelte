<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import Building2Icon from '@lucide/svelte/icons/building-2';
	import CreditCardIcon from '@lucide/svelte/icons/credit-card';
	import MonitorIcon from '@lucide/svelte/icons/monitor';
	import SearchIcon from '@lucide/svelte/icons/search';
	import UserCheckIcon from '@lucide/svelte/icons/user-check';
	import UsersIcon from '@lucide/svelte/icons/users';
	import { api, ApiError } from '#lib/api.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Empty from '#lib/components/ui/empty/index.js';
	import * as InputGroup from '#lib/components/ui/input-group/index.js';
	import * as Item from '#lib/components/ui/item/index.js';
	import * as Pagination from '#lib/components/ui/pagination/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import { config } from '#lib/config.js';
	import { formatDayTime } from '#lib/format.js';
	import { notifyError, notifyFailure, notifySuccess } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { AdminDashboard, AdminUser } from '#lib/types.js';
	import MetricCard from './metric-card.svelte';

	let { dashboard }: { dashboard: AdminDashboard } = $props();

	const users = $derived(dashboard.users);
	const currentUserId = $derived(session.account?.user.id);

	let revoking = $state<AdminUser | null>(null);
	let confirmOpen = $state(false);
	let pending = $state(false);

	const actionLabels: Record<string, string> = {
		'admin.impersonation_started': 'Impersonation started',
		'admin.impersonation_stopped': 'Impersonation stopped',
		'admin.sessions_revoked': 'User sessions revoked',
		'workspace.deleted': 'Workspace deleted',
		'subscription.updated': 'Subscription updated'
	};

	function pageURL(page: number): string {
		const search = new URLSearchParams();
		if (users.query) {
			search.set('q', users.query);
		}
		search.set('page', String(page));
		return `/admin?${search}`;
	}

	function confirmRevoke(user: AdminUser) {
		revoking = user;
		confirmOpen = true;
	}

	async function revoke() {
		if (!revoking || pending) {
			return;
		}
		pending = true;
		try {
			await api.post(`/api/v1/admin/users/${encodeURIComponent(revoking.id)}/sessions/revoke`);
			notifySuccess('Sessions revoked', 'The user has been signed out everywhere.');
		} catch (cause) {
			notifyRevokeFailure(cause);
		}
		confirmOpen = false;
		pending = false;
		await invalidate((url) => url.pathname === '/api/v1/admin/users');
	}

	function notifyRevokeFailure(cause: unknown) {
		if (cause instanceof ApiError && cause.status === 404) {
			notifyError('User not found', 'The requested user or session no longer exists.');
		} else if (cause instanceof ApiError && (cause.code === 'admin_self_action' || cause.status === 403)) {
			notifyError('Action not allowed', cause.message);
		} else {
			notifyFailure(cause, 'Admin action failed', 'The admin action could not be completed. Please try again.');
		}
	}
</script>

<div class="space-y-8">
	<div>
		<h1 class="text-3xl font-bold tracking-tight">Admin</h1>
		<p class="mt-2 text-muted-foreground">
			{config.workspaces
				? 'Users, workspaces, sessions, and operator activity.'
				: 'Users, subscriptions, sessions, and operator activity.'}
		</p>
	</div>
	<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-5">
		<MetricCard label="Users" value={dashboard.metrics.users} Icon={UsersIcon} />
		<MetricCard label="Verified" value={dashboard.metrics.verified_users} Icon={UserCheckIcon} />
		{#if config.workspaces}
			<MetricCard label="Workspaces" value={dashboard.metrics.workspaces ?? 0} Icon={Building2Icon} />
			<MetricCard label="Paid" value={dashboard.metrics.paid_workspaces ?? 0} Icon={CreditCardIcon} />
		{:else}
			<MetricCard label="Accounts" value={dashboard.metrics.users} Icon={UsersIcon} />
			<MetricCard label="Paid" value={dashboard.metrics.paid_users ?? 0} Icon={CreditCardIcon} />
		{/if}
		<MetricCard label="Sessions" value={dashboard.metrics.active_sessions} Icon={MonitorIcon} />
	</div>
	<Card.Root>
		<Card.Header>
			<Card.Title>Users</Card.Title>
			<Card.Description>Search accounts and revoke their sessions.</Card.Description>
		</Card.Header>
		<Card.Content>
			<form method="GET" action="/admin" class="mb-5 flex gap-2" role="search" data-sveltekit-reset="false">
				<InputGroup.Root class="max-w-md flex-1">
					<InputGroup.Addon>
						<SearchIcon />
					</InputGroup.Addon>
					<InputGroup.Input
						type="search"
						name="q"
						value={users.query}
						placeholder="Search name or email"
						aria-label="Search name or email"
					/>
				</InputGroup.Root>
				<Button type="submit" variant="outline">Search</Button>
			</form>
			<div class="rounded-lg border">
				<Table.Root>
					<Table.Header>
						<Table.Row>
							<Table.Head class="px-4">User</Table.Head>
							<Table.Head class="px-4">Status</Table.Head>
							{#if config.workspaces}
								<Table.Head class="px-4">Workspaces</Table.Head>
							{/if}
							<Table.Head class="px-4">Sessions</Table.Head>
							<Table.Head class="px-4 text-right">Actions</Table.Head>
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each users.items as user (user.id)}
							<Table.Row>
								<Table.Cell class="px-4 py-3">
									<div class="font-medium">{user.name}</div>
									<div class="text-muted-foreground">{user.email}</div>
								</Table.Cell>
								<Table.Cell class="px-4 py-3">
									<div class="flex flex-wrap gap-1.5">
										{#if user.email_verified_at}
											<Badge variant="secondary">Verified</Badge>
										{:else}
											<Badge variant="outline">Unverified</Badge>
										{/if}
										{#if user.is_admin}
											<Badge>Operator</Badge>
										{/if}
									</div>
								</Table.Cell>
								{#if config.workspaces}
									<Table.Cell class="px-4 py-3">{user.workspace_count ?? 0}</Table.Cell>
								{/if}
								<Table.Cell class="px-4 py-3">{user.active_session_count}</Table.Cell>
								<Table.Cell class="px-4 py-3">
									<div class="flex justify-end gap-2">
										{#if user.id !== currentUserId}
											<Button
												variant="outline"
												size="sm"
												aria-haspopup="dialog"
												aria-label="Revoke sessions of {user.email}"
												onclick={() => confirmRevoke(user)}
											>
												Revoke sessions
											</Button>
										{/if}
									</div>
								</Table.Cell>
							</Table.Row>
						{:else}
							<Table.Row>
								<Table.Cell colspan={config.workspaces ? 5 : 4} class="px-4 py-6 text-center text-muted-foreground">
									No users found.
								</Table.Cell>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			</div>
			<div class="mt-4 flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
				<span>{users.total_users} users</span>
				<Pagination.Root
					count={Math.max(users.total_pages, 1)}
					perPage={1}
					page={users.page}
					onPageChange={(page) => goto(pageURL(page), { reset: false })}
					class="mx-0 w-auto"
				>
					<Pagination.Content>
						<Pagination.Item>
							<Pagination.Previous />
						</Pagination.Item>
						<Pagination.Item class="px-2">Page {users.page} of {users.total_pages}</Pagination.Item>
						<Pagination.Item>
							<Pagination.Next />
						</Pagination.Item>
					</Pagination.Content>
				</Pagination.Root>
			</div>
		</Card.Content>
	</Card.Root>
	<Card.Root>
		<Card.Header>
			<Card.Title>Operator activity</Card.Title>
			<Card.Description>
				{config.workspaces
					? 'Security, workspace deletion, and subscription events retained for 180 days.'
					: 'Security and subscription events retained for 180 days.'}
			</Card.Description>
		</Card.Header>
		<Card.Content>
			{#if dashboard.events.length === 0}
				<Empty.Root>
					<Empty.Header>
						<Empty.Media variant="icon">
							<ActivityIcon />
						</Empty.Media>
						<Empty.Title>No operator activity yet</Empty.Title>
						<Empty.Description>Security and administrative events will appear here.</Empty.Description>
					</Empty.Header>
				</Empty.Root>
			{:else}
				<Item.Group class="gap-2">
					{#each dashboard.events as event (event.id)}
						<Item.Root size="sm">
							<Item.Content>
								<Item.Title>{actionLabels[event.action] ?? event.action}</Item.Title>
								<Item.Description>
									{event.actor_email || 'System'}
									<span>· {event.target_type} {event.target_id}</span>
								</Item.Description>
							</Item.Content>
							<Item.Actions>
								<time class="text-sm text-muted-foreground" datetime={event.created_at}>
									{formatDayTime(event.created_at)}
								</time>
							</Item.Actions>
						</Item.Root>
					{/each}
				</Item.Group>
			{/if}
		</Card.Content>
	</Card.Root>
</div>

<AlertDialog.Root bind:open={confirmOpen}>
	<AlertDialog.Content>
		<AlertDialog.Header>
			<AlertDialog.Title>Revoke sessions</AlertDialog.Title>
			<AlertDialog.Description>
				Revoke every active session for {revoking?.email ?? 'this user'}?
			</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action onclick={revoke} aria-busy={pending || undefined}>Revoke sessions</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
