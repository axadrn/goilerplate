<script lang="ts">
	import { goto, invalidate } from '$app/navigation';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { session } from '#lib/session.svelte.js';
	import type { WorkspaceDetail } from '#lib/types.js';
	import { firstWorkspacePath } from '#lib/workspace/workspace.js';
	import GeneralCard from './general-card.svelte';
	import InvitationsCard from './invitations-card.svelte';
	import InviteCard from './invite-card.svelte';
	import LeaveDeleteCard from './leave-delete-card.svelte';
	import MembersCard from './members-card.svelte';

	let { detail }: { detail: WorkspaceDetail } = $props();

	const apiPath = $derived(`/api/v1/workspaces/${encodeURIComponent(detail.workspace.slug)}`);
	const seats = $derived(
		detail.seats.limit === null ? `${detail.seats.used} seats` : `${detail.seats.used} of ${detail.seats.limit} seats`
	);

	// reload fetches the workspace again after a change; the sidebar shows
	// names and roles from the account.
	async function reload() {
		await Promise.all([
			invalidate((url) => url.pathname === apiPath),
			session.refresh().catch(() => null)
		]);
	}

	// leave opens another workspace once this one is no longer the
	// person's, like the HTML app.
	async function leave() {
		const account = await session.refresh().catch(() => null);
		await goto(firstWorkspacePath(account));
	}
</script>

<div class="container max-w-5xl space-y-6 px-6 py-8">
	<div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
		<div>
			<h1 class="text-3xl font-bold">Workspace</h1>
			<p class="mt-2 text-muted-foreground">
				Manage {detail.workspace.name}, its settings, members, and invitations.
			</p>
		</div>
		<Badge variant="secondary">{seats}</Badge>
	</div>
	{#if detail.permissions.manage_workspace}
		<GeneralCard {apiPath} name={detail.workspace.name} onchange={reload} />
	{/if}
	{#if detail.permissions.manage_members}
		<InviteCard {apiPath} onchange={reload} />
	{/if}
	<MembersCard {apiPath} {detail} onchange={reload} onleave={leave} />
	{#if detail.permissions.manage_members && detail.invitations.length > 0}
		<InvitationsCard {apiPath} invitations={detail.invitations} onchange={reload} />
	{/if}
	<LeaveDeleteCard {apiPath} {detail} onleave={leave} />
</div>
