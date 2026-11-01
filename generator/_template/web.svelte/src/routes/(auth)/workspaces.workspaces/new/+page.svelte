<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { notifyFailure } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { Membership } from '#lib/types.js';

	let name = $state('');
	let message = $state('');
	let pending = $state(false);
	let input = $state<HTMLInputElement | null>(null);

	onMount(() => input?.focus());

	async function fail(text: string) {
		message = text;
		pending = false;
		await tick();
		input?.focus();
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		if (!name.trim()) {
			await fail('Enter a workspace name');
			return;
		}
		pending = true;
		let created: Membership;
		try {
			created = await api.post<Membership>('/api/v1/workspaces', { body: { name: name.trim() } });
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 422) {
				await fail('Enter a workspace name');
				return;
			}
			pending = false;
			notifyFailure(cause, 'Error', 'The workspace could not be created. Please try again.');
			return;
		}
		await session.refresh().catch(() => null);
		await goto(`/app/${created.workspace.slug}/dashboard`);
	}
</script>

<AuthCard title="New workspace" description="Create a separate space for your team">
	<form id="new-workspace-form" method="POST" onsubmit={submit} aria-busy={pending || undefined} inert={pending}>
		<Field.Group>
			<Field.Field>
				<Field.Label for="workspace-name">Workspace name</Field.Label>
				<Input
					bind:ref={input}
					bind:value={name}
					id="workspace-name"
					name="name"
					type="text"
					placeholder="Acme"
					aria-invalid={message ? 'true' : 'false'}
					aria-describedby={message ? 'workspace-name-error' : undefined}
				/>
				{#if message}
					<Field.Error id="workspace-name-error">{message}</Field.Error>
				{/if}
			</Field.Field>
			<Button type="submit" class="w-full">Create workspace</Button>
		</Field.Group>
	</form>
</AuthCard>
