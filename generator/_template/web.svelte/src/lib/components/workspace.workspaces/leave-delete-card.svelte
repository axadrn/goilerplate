<script lang="ts">
	import { api } from '#lib/api.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { buttonVariants } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import type { WorkspaceDetail } from '#lib/types.js';
	import { notifyWorkspaceFailure } from '#lib/workspace/workspace.js';

	let { apiPath, detail, onleave }: { apiPath: string; detail: WorkspaceDetail; onleave: () => Promise<void> } =
		$props();

	let leaveOpen = $state(false);
	let deleteOpen = $state(false);
	let pending = $state(false);

	async function run(request: () => Promise<unknown>) {
		if (pending) {
			return;
		}
		pending = true;
		try {
			await request();
		} catch (cause) {
			pending = false;
			leaveOpen = false;
			deleteOpen = false;
			notifyWorkspaceFailure(cause);
			return;
		}
		leaveOpen = false;
		deleteOpen = false;
		pending = false;
		await onleave();
	}
</script>

<Card.Root class="border-destructive/30">
	<Card.Header>
		<Card.Title>Leave or delete workspace</Card.Title>
		<Card.Description>The last Owner must transfer ownership or delete the workspace.</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="flex flex-wrap gap-3">
			<AlertDialog.Root bind:open={leaveOpen}>
				<AlertDialog.Trigger class={buttonVariants({ variant: 'outline' })}>Leave workspace</AlertDialog.Trigger>
				<AlertDialog.Content>
					<AlertDialog.Header>
						<AlertDialog.Title>Leave {detail.workspace.name}?</AlertDialog.Title>
						<AlertDialog.Description>
							You will lose access to this workspace. A sole Owner must transfer ownership first.
						</AlertDialog.Description>
					</AlertDialog.Header>
					<AlertDialog.Footer>
						<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
						<AlertDialog.Action
							class={buttonVariants({ variant: 'destructive' })}
							aria-busy={pending || undefined}
							onclick={() => run(() => api.post(`${apiPath}/leave`))}
						>
							Leave workspace
						</AlertDialog.Action>
					</AlertDialog.Footer>
				</AlertDialog.Content>
			</AlertDialog.Root>
			{#if detail.permissions.delete_workspace}
				<AlertDialog.Root bind:open={deleteOpen}>
					<AlertDialog.Trigger class={buttonVariants({ variant: 'destructive' })}>Delete workspace</AlertDialog.Trigger>
					<AlertDialog.Content>
						<AlertDialog.Header>
							<AlertDialog.Title>Permanently delete {detail.workspace.name}?</AlertDialog.Title>
							<AlertDialog.Description>
								This deletes the workspace, members, invitations, audit history, and free subscription record. Pending
								checkouts must expire. Paid subscriptions must be cancelled and fully expired first. This cannot be undone.
							</AlertDialog.Description>
						</AlertDialog.Header>
						<AlertDialog.Footer>
							<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
							<AlertDialog.Action
								class={buttonVariants({ variant: 'destructive' })}
								aria-busy={pending || undefined}
								onclick={() => run(() => api.delete(apiPath))}
							>
								Delete permanently
							</AlertDialog.Action>
						</AlertDialog.Footer>
					</AlertDialog.Content>
				</AlertDialog.Root>
			{/if}
		</div>
	</Card.Content>
</Card.Root>
