<script lang="ts">
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import * as AlertDialog from '#lib/components/ui/alert-dialog/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { notifyError, notifyFailure, notifySuccess } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { User } from '#lib/types.js';

	// Changing, setting or removing the password signs out every other
	// session; onsessions lets the page reload its session list.
	let { user, onsessions }: { user: User; onsessions: () => void } = $props();

	type Field = 'current_password' | 'new_password' | 'confirm_password';

	let current = $state('');
	let next = $state('');
	let confirm = $state('');
	let invalid = $state<Field | null>(null);
	let pending = $state(false);
	let removeOpen = $state(false);
	let removing = $state(false);
	let inputs = $state<Record<Field, HTMLInputElement | null>>({
		current_password: null,
		new_password: null,
		confirm_password: null
	});

	async function reject(field: Field, title: string, description: string) {
		invalid = field;
		notifyError(title, description);
		await tick();
		inputs[field]?.focus();
	}

	async function done(description: string) {
		current = '';
		next = '';
		confirm = '';
		invalid = null;
		notifySuccess('Success', description);
		// The account says whether a password is set now.
		await session.refresh().catch(() => null);
		onsessions();
	}

	async function change(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		if (!current || !next || !confirm) {
			await reject(!current ? 'current_password' : !next ? 'new_password' : 'confirm_password', 'Error', 'All password fields are required');
			return;
		}
		if (next !== confirm) {
			await reject('confirm_password', 'Error', 'New passwords do not match');
			return;
		}
		pending = true;
		try {
			await api.post('/api/v1/account/password', { body: { current_password: current, new_password: next } });
			await done('Password updated successfully');
		} catch (cause) {
			if (cause instanceof ApiError && cause.fields.current_password) {
				await reject('current_password', 'Error', 'Current password is incorrect');
			} else if (cause instanceof ApiError && cause.fields.new_password) {
				await reject('new_password', 'Error', cause.fields.new_password);
			} else {
				notifyFailure(cause, 'Error', 'Failed to update password');
			}
		} finally {
			pending = false;
		}
	}

	async function set(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		if (!next || !confirm) {
			await reject(!next ? 'new_password' : 'confirm_password', 'Error', 'All fields are required');
			return;
		}
		if (next !== confirm) {
			await reject('confirm_password', 'Error', 'Passwords do not match');
			return;
		}
		pending = true;
		try {
			await api.post('/api/v1/account/password/set', { body: { password: next } });
			await done('Password set successfully. You can now sign in with your password.');
			// The change form replaces this one.
			await tick();
			inputs.current_password?.focus();
		} catch (cause) {
			if (cause instanceof ApiError && cause.fields.password) {
				await reject('new_password', 'Error', cause.fields.password);
			} else {
				notifyFailure(cause, 'Failed to Set Password');
			}
		} finally {
			pending = false;
		}
	}

	async function remove() {
		if (removing) {
			return;
		}
		removing = true;
		try {
			await api.delete('/api/v1/account/password');
			await done('Password removed. You can now only sign in with magic links.');
			removeOpen = false;
		} catch (cause) {
			removeOpen = false;
			notifyFailure(cause, 'Failed to Remove Password');
		} finally {
			removing = false;
		}
	}
</script>

{#snippet passwordInput(field: Field, placeholder: string, autocomplete: 'current-password' | 'new-password')}
	{#if field === 'current_password'}
		<Input
			bind:ref={inputs.current_password}
			bind:value={current}
			type="password"
			id={field}
			name={field}
			{placeholder}
			{autocomplete}
			aria-invalid={invalid === field ? 'true' : 'false'}
		/>
	{:else if field === 'new_password'}
		<Input
			bind:ref={inputs.new_password}
			bind:value={next}
			type="password"
			id={field}
			name={field}
			{placeholder}
			{autocomplete}
			aria-invalid={invalid === field ? 'true' : 'false'}
			aria-describedby="new-password-hint"
		/>
	{:else}
		<Input
			bind:ref={inputs.confirm_password}
			bind:value={confirm}
			type="password"
			id={field}
			name={field}
			{placeholder}
			{autocomplete}
			aria-invalid={invalid === field ? 'true' : 'false'}
		/>
	{/if}
{/snippet}

<Card.Root>
	<Card.Header>
		<Card.Title>Password</Card.Title>
		<Card.Description>
			{user.password_set
				? 'You can sign in with your password or magic links'
				: "You're using passwordless login (magic links only)"}
		</Card.Description>
	</Card.Header>
	<Card.Content>
		<div id="password-form" class="space-y-4">
			{#if user.password_set}
				<form class="space-y-4" onsubmit={change} aria-busy={pending || undefined}>
					<div class="space-y-2">
						<Label for="current_password">Current Password</Label>
						{@render passwordInput('current_password', 'Enter current password', 'current-password')}
					</div>
					<div class="space-y-2">
						<Label for="new_password">New Password</Label>
						{@render passwordInput('new_password', 'Enter new password', 'new-password')}
						<p id="new-password-hint" class="text-sm text-muted-foreground">Must be at least 12 characters long</p>
					</div>
					<div class="space-y-2">
						<Label for="confirm_password">Confirm New Password</Label>
						{@render passwordInput('confirm_password', 'Confirm new password', 'new-password')}
					</div>
					<div class="flex items-center justify-between">
						<Button variant="outline" aria-haspopup="dialog" onclick={() => (removeOpen = true)}>Remove Password</Button>
						<Button type="submit">Update Password</Button>
					</div>
				</form>
			{:else}
				<div class="space-y-4">
					<div class="rounded-lg border bg-muted/50 p-4">
						<p class="text-sm text-muted-foreground">
							Set a password to enable password-based sign in. You can still use magic links anytime.
						</p>
					</div>
					<form class="space-y-4" onsubmit={set} aria-busy={pending || undefined}>
						<div class="space-y-2">
							<Label for="new_password">New Password</Label>
							{@render passwordInput('new_password', 'Enter new password', 'new-password')}
							<p id="new-password-hint" class="text-sm text-muted-foreground">Must be at least 12 characters long</p>
						</div>
						<div class="space-y-2">
							<Label for="confirm_password">Confirm Password</Label>
							{@render passwordInput('confirm_password', 'Confirm new password', 'new-password')}
						</div>
						<div class="flex justify-end">
							<Button type="submit">Set Password</Button>
						</div>
					</form>
				</div>
			{/if}
		</div>
	</Card.Content>
</Card.Root>

<!-- Outside the form: removing the password replaces the form while the dialog is still open. -->
<AlertDialog.Root bind:open={removeOpen}>
	<AlertDialog.Content
		onCloseAutoFocus={(event) => {
			if (!user.password_set) {
				event.preventDefault();
				inputs.new_password?.focus();
			}
		}}
	>
		<AlertDialog.Header>
			<AlertDialog.Title>Remove Password</AlertDialog.Title>
			<AlertDialog.Description>Are you sure? You'll only be able to sign in with magic links.</AlertDialog.Description>
		</AlertDialog.Header>
		<AlertDialog.Footer>
			<AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
			<AlertDialog.Action onclick={remove} aria-busy={removing || undefined}>Remove Password</AlertDialog.Action>
		</AlertDialog.Footer>
	</AlertDialog.Content>
</AlertDialog.Root>
