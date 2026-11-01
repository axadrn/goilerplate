<script lang="ts">
	import { api } from '#lib/api.js';
	import { Badge } from '#lib/components/ui/badge/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { session } from '#lib/session.svelte.js';
	import type { WorkspaceMember, WorkspacePermissions, WorkspaceRole } from '#lib/types.js';
	import { notifyWorkspaceFailure, notifyWorkspaceUpdated, roleLabels } from '#lib/workspace/workspace.js';

	let {
		apiPath,
		member,
		permissions,
		onchange,
		onleave
	}: {
		apiPath: string;
		member: WorkspaceMember;
		permissions: WorkspacePermissions;
		onchange: () => Promise<void>;
		onleave: () => Promise<void>;
	} = $props();

	// Owners change only for members who may manage owners, like the HTML page.
	const manageable = $derived(permissions.manage_members && (member.role !== 'owner' || permissions.manage_owners));
	const roleItems = $derived(
		(permissions.manage_owners ? (['owner', 'admin', 'member'] as const) : (['admin', 'member'] as const)).map(
			(value) => ({ value, label: roleLabels[value] })
		)
	);
	const memberPath = $derived(`${apiPath}/members/${encodeURIComponent(member.id)}`);

	let role = $derived<string>(member.role);
	let pending = $state<'role' | 'remove' | null>(null);

	async function saveRole(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		pending = 'role';
		try {
			await api.patch(`${memberPath}/role`, { body: { role } });
			notifyWorkspaceUpdated('Member role updated.');
			await onchange();
		} catch (cause) {
			role = member.role;
			notifyWorkspaceFailure(cause);
		} finally {
			pending = null;
		}
	}

	async function remove() {
		if (pending) {
			return;
		}
		pending = 'remove';
		try {
			await api.delete(memberPath);
		} catch (cause) {
			pending = null;
			notifyWorkspaceFailure(cause);
			return;
		}
		pending = null;
		if (member.user_id === session.account?.user.id) {
			await onleave();
			return;
		}
		notifyWorkspaceUpdated('Member removed.');
		await onchange();
	}

	function label(value: WorkspaceRole): string {
		return roleLabels[value];
	}
</script>

<li class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
	<div class="min-w-0">
		<p class="truncate font-medium">{member.name}</p>
		{#if member.email}
			<p class="truncate text-sm text-muted-foreground">{member.email}</p>
		{/if}
	</div>
	<div class="flex flex-wrap items-center gap-2">
		{#if manageable}
			<form method="POST" class="flex items-center gap-2" onsubmit={saveRole} aria-busy={pending === 'role' || undefined}>
				<Select.Root type="single" name="role" bind:value={role} items={roleItems}>
					<Select.Trigger class="w-32" aria-label="Role of {member.name}">
						<Select.Value />
					</Select.Trigger>
					<Select.Content align="start">
						{#each roleItems as item (item.value)}
							<Select.Item value={item.value} label={item.label} />
						{/each}
					</Select.Content>
				</Select.Root>
				<Button type="submit" variant="outline" size="sm" aria-label="Save role of {member.name}">Save</Button>
			</form>
			<Button
				variant="ghost"
				size="sm"
				aria-label="Remove {member.name}"
				aria-busy={pending === 'remove' || undefined}
				onclick={remove}
			>
				Remove
			</Button>
		{:else}
			<Badge>{label(member.role)}</Badge>
		{/if}
	</div>
</li>
