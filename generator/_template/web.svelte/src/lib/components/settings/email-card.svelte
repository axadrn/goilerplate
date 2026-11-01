<script lang="ts">
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Card from '#lib/components/ui/card/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { notifyError, notifyFailure, notifyInfo } from '#lib/notify.js';
	import type { User } from '#lib/types.js';

	let { user }: { user: User } = $props();

	let email = $derived(user.email);
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
		const value = email.trim().toLowerCase();
		if (!value) {
			await reject('Error', 'Valid email is required');
			return;
		}
		if (value === user.email.toLowerCase()) {
			invalid = false;
			notifyInfo('Info', 'Email address is already set to this value');
			return;
		}
		pending = true;
		try {
			// 202: the change waits for the link sent to the new address.
			await api.patch('/api/v1/account/email', { body: { email: value } });
			invalid = false;
			notifyInfo('Verification Required', 'Please check your new email address to verify the change');
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 422) {
				await reject('Error', 'Valid email is required');
			} else if (cause instanceof ApiError && cause.code === 'email_taken') {
				await reject('Error', 'Email already in use');
			} else {
				notifyFailure(cause, 'Error', 'Failed to change email');
			}
		} finally {
			pending = false;
		}
	}
</script>

<Card.Root>
	<Card.Header>
		<Card.Title>Email Address</Card.Title>
		<Card.Description>Update your email address</Card.Description>
	</Card.Header>
	<Card.Content>
		<form class="space-y-4" onsubmit={submit} aria-busy={pending || undefined}>
			<div class="space-y-2">
				<Label for="email">Email</Label>
				<Input
					bind:ref={input}
					bind:value={email}
					type="email"
					id="email"
					name="email"
					placeholder="your@email.com"
					autocomplete="email"
					aria-invalid={invalid ? 'true' : 'false'}
				/>
			</div>
			<div class="flex justify-end">
				<Button type="submit">Update Email</Button>
			</div>
		</form>
	</Card.Content>
</Card.Root>
