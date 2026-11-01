<script lang="ts">
	import Trash2Icon from '@lucide/svelte/icons/trash-2';
	import { api, ApiError } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Dialog from '#lib/components/ui/dialog/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { config } from '#lib/config.js';
	import { leaveApp } from '#lib/logout.js';
	import { notifyError, notifyFailure } from '#lib/notify.js';
	import type { User } from '#lib/types.js';

	let { user }: { user: User } = $props();

	let open = $state(false);
	let confirmation = $state('');
	let pending = $state(false);

	// The server asks for the account's email address, like the API.
	const confirmed = $derived(confirmation.trim().toLowerCase() === user.email.toLowerCase());

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending || !confirmed) {
			return;
		}
		pending = true;
		try {
			await api.delete('/api/v1/account', { body: { confirm_email: confirmation.trim() } });
		} catch (cause) {
			pending = false;
			open = false;
			confirmation = '';
			if (cause instanceof ApiError && cause.code === 'active_subscription') {
				notifyError(
					'Active Subscription',
					'Please cancel your subscription before deleting your account. Visit billing to manage your subscription.'
				);
			} else if (cause instanceof ApiError && cause.code === 'sole_workspace_owner') {
				notifyError(
					'Workspace ownership required',
					'Transfer ownership or delete workspaces where you are the only owner before deleting your account.'
				);
			} else {
				notifyFailure(cause, 'Error', 'Failed to delete account. Please try again.');
			}
			return;
		}
		leaveApp();
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Danger Zone</Card.Title>
		<Card.Description>Permanently delete your account and all associated data</Card.Description>
	</Card.Header>
	<Card.Content>
		<div class="space-y-4">
			<div class="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
				<h3 class="mb-2 font-semibold text-destructive">Delete Account</h3>
				<p class="mb-4 text-sm text-muted-foreground">
					Once you delete your account, there is no going back. Please be certain.
				</p>
				<Dialog.Root bind:open onOpenChange={(next) => next || (confirmation = '')}>
					<Dialog.Trigger>
						{#snippet child({ props })}
							<Button variant="destructive" {...props}>
								<Trash2Icon class="mr-2 size-4" />
								Delete Account
							</Button>
						{/snippet}
					</Dialog.Trigger>
					<Dialog.Content>
						<Dialog.Header>
							<Dialog.Title>Delete Account</Dialog.Title>
							<Dialog.Description>
								{config.storage
									? 'This will permanently delete your account and all associated data including your profile, files, and settings. This action cannot be undone.'
									: 'This will permanently delete your account and all associated data including your profile and settings. This action cannot be undone.'}
							</Dialog.Description>
						</Dialog.Header>
						<form id="delete-account-form" onsubmit={submit} aria-busy={pending || undefined}>
							<div class="space-y-2 py-4">
								<Label for="delete-confirmation">
									<span>Type <span class="font-mono font-semibold">{user.email}</span> to confirm</span>
								</Label>
								<Input
									bind:value={confirmation}
									type="email"
									id="delete-confirmation"
									name="confirm_email"
									placeholder={user.email}
									autocomplete="off"
								/>
							</div>
							<Dialog.Footer>
								<Dialog.Close>
									{#snippet child({ props })}
										<Button variant="outline" {...props}>Cancel</Button>
									{/snippet}
								</Dialog.Close>
								<Button type="submit" disabled={!confirmed}>Yes, Delete My Account</Button>
							</Dialog.Footer>
						</form>
					</Dialog.Content>
				</Dialog.Root>
			</div>
		</div>
	</Card.Content>
</Card.Root>
