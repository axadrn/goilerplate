<script lang="ts">
	import { tick } from 'svelte';
	import { api } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { formatDay } from '#lib/format.js';
	import type { PendingInvitation } from '#lib/types.js';
	import { notifyWorkspaceFailure, notifyWorkspaceUpdated, roleLabels } from '#lib/workspace/workspace.js';

	let {
		apiPath,
		invitations,
		onchange
	}: { apiPath: string; invitations: PendingInvitation[]; onchange: () => Promise<void> } = $props();

	let pending = $state<string | null>(null);

	async function act(invitation: PendingInvitation, kind: 'resend' | 'revoke') {
		if (pending) {
			return;
		}
		pending = `${kind}:${invitation.id}`;
		const path = `${apiPath}/invitations/${encodeURIComponent(invitation.id)}`;
		try {
			if (kind === 'resend') {
				await api.post(`${path}/resend`);
				notifyWorkspaceUpdated('Invitation sent again.');
			} else {
				await api.delete(path);
				notifyWorkspaceUpdated('Invitation revoked.');
			}
			await onchange();
		} catch (cause) {
			notifyWorkspaceFailure(cause);
		} finally {
			pending = null;
		}
		// A revoked row and its buttons are gone; keep focus on the page.
		await tick();
		if (!document.activeElement || document.activeElement === document.body) {
			document.querySelector<HTMLElement>('#invite-email')?.focus();
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Pending invitations</Card.Title>
	</Card.Header>
	<Card.Content>
		<ul class="divide-y rounded-lg border" aria-label="Pending invitations">
			{#each invitations as invitation (invitation.id)}
				<li class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
					<div class="min-w-0">
						<p class="truncate font-medium">{invitation.email}</p>
						<p class="text-sm text-muted-foreground">
							{roleLabels[invitation.role]}, expires {formatDay(invitation.expires_at)}
						</p>
					</div>
					<div class="flex gap-2">
						<Button
							variant="outline"
							size="sm"
							aria-label="Resend invitation to {invitation.email}"
							aria-busy={pending === `resend:${invitation.id}` || undefined}
							onclick={() => act(invitation, 'resend')}
						>
							Resend
						</Button>
						<Button
							variant="ghost"
							size="sm"
							aria-label="Revoke invitation to {invitation.email}"
							aria-busy={pending === `revoke:${invitation.id}` || undefined}
							onclick={() => act(invitation, 'revoke')}
						>
							Revoke
						</Button>
					</div>
				</li>
			{/each}
		</ul>
	</Card.Content>
</Card.Root>
