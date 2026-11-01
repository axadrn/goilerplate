<script lang="ts">
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { notifyError, notifyFailure, notifySuccess } from '#lib/notify.js';
	import { session } from '#lib/session.svelte.js';
	import type { User } from '#lib/types.js';

	let { user }: { user: User } = $props();

	// Starts with the saved name; typing changes only this field.
	let name = $derived(user.name);
	let invalid = $state(false);
	let pending = $state(false);
	let input = $state<HTMLInputElement | null>(null);

	async function reject(title: string, description: string) {
		invalid = true;
		notifyError(title, description);
		await tick();
		input?.focus();
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (pending) {
			return;
		}
		if (!name.trim()) {
			await reject('Error', 'Name is required');
			return;
		}
		pending = true;
		try {
			const answer = await api.patch<{ user: User }>('/api/v1/account/name', { body: { name } });
			if (session.account) {
				session.account.user = answer.user;
			}
			invalid = false;
			notifySuccess('Success', 'Name updated successfully');
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 422) {
				await reject('Error', 'Failed to update name');
			} else {
				notifyFailure(cause, 'Error', 'Failed to update name');
			}
		} finally {
			pending = false;
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Name</Card.Title>
		<Card.Description>Update your display name</Card.Description>
	</Card.Header>
	<Card.Content>
		<form class="space-y-4" onsubmit={submit} aria-busy={pending || undefined}>
			<div class="space-y-2">
				<Label for="name">Name</Label>
				<Input
					bind:ref={input}
					bind:value={name}
					type="text"
					id="name"
					name="name"
					placeholder="Your name"
					autocomplete="name"
					aria-invalid={invalid ? 'true' : 'false'}
				/>
			</div>
			<div class="flex justify-end">
				<Button type="submit">Save Name</Button>
			</div>
		</form>
	</Card.Content>
</Card.Root>
