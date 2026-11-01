<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import UsersIcon from '@lucide/svelte/icons/users';
	import { onMount, tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { config } from '#lib/config.js';
	import { notifyFailure } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { Membership } from '#lib/types.js';
	import { roleLabels } from '#lib/workspace/workspace.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const token = $derived(page.params.token ?? '');
	let preview = $derived(data.preview);
	let error = $derived(data.error);
	let pending = $state(false);
	let alert = $state<HTMLElement | null>(null);

	// The preview request sets the pending_invitation cookie for guests and
	// accounts without a name, so sign-in and onboarding return here, like
	// the HTML app.
	onMount(() => {
		if (data.preview && data.account?.onboarding_required) {
			goto('/auth/onboarding', { replace: true });
		}
	});

	async function accept(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		pending = true;
		let membership: Membership;
		try {
			membership = await api.post<Membership>(`/api/v1/invitations/${encodeURIComponent(token)}/accept`);
		} catch (cause) {
			pending = false;
			const message = acceptError(cause);
			if (!message) {
				notifyFailure(cause, 'Error', 'The invitation could not be accepted. Please try again.');
				return;
			}
			error = message;
			if (cause instanceof ApiError && (cause.status === 410 || cause.status === 404)) {
				preview = null;
			}
			await tick();
			alert?.focus();
			return;
		}
		await session.refresh().catch(() => null);
		await goto(`/app/${membership.workspace.slug}/dashboard`);
	}

	function acceptError(cause: unknown): string {
		if (!(cause instanceof ApiError)) {
			return '';
		}
		switch (cause.code) {
			case 'invitation_unavailable':
				return 'This invitation is no longer available.';
			case 'not_found':
				return 'This invitation is invalid or no longer available.';
			case 'invitation_email_mismatch':
				return 'Sign in with the email address that received this invitation.';
			case 'seat_limit':
				return 'This workspace has no available seats. Ask an owner to upgrade or remove a member.';
			case 'already_member':
				return 'You already belong to this workspace.';
		}
		return '';
	}
</script>

<Card.Root>
	<Card.Header>
		<div class="mb-3 flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
			<UsersIcon class="size-5" />
		</div>
		<Card.Title>Workspace invitation</Card.Title>
		<Card.Description>
			{#if preview}
				{preview.inviter_name} invited you to join {preview.workspace_name} as {roleLabels[preview.role]}.
			{:else}
				This invitation is not available.
			{/if}
		</Card.Description>
	</Card.Header>
	{#if error || preview}
		<Card.Content>
			{#if error}
				<div
					bind:this={alert}
					role="alert"
					tabindex="-1"
					class="mb-4 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive outline-none"
				>
					{error}
				</div>
			{/if}
			{#if preview && data.account && !data.account.onboarding_required}
				<div class="mb-5 rounded-lg border bg-muted/40 p-4 text-sm">
					<p class="font-medium">Signed in as {data.account.user.email}</p>
					<p class="mt-1 text-muted-foreground">This must be the address that received the invitation.</p>
				</div>
				<form method="POST" onsubmit={accept} aria-busy={pending || undefined} inert={pending}>
					<Button type="submit" class="w-full">Join {preview.workspace_name}</Button>
				</form>
			{:else if preview && !data.account}
				<p class="mb-5 text-sm text-muted-foreground">
					Sign in with the email address that received this invitation to join.
				</p>
				<Button href="/auth" class="w-full">Sign in to join</Button>
			{/if}
		</Card.Content>
	{/if}
	<Card.Footer>
		<Button href="/" variant="ghost" class="w-full">Back to {config.name}</Button>
	</Card.Footer>
</Card.Root>
