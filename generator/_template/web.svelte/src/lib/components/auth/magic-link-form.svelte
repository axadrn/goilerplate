<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount, tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import { authEmail } from '#lib/auth-email.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { notifyFailure } from '#lib/notify.js';

	let { error = '' }: { error?: string } = $props();

	let email = $state('');
	let message = $derived(error);
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
		const value = email.trim();
		if (!value) {
			await fail('Email is required');
			return;
		}
		pending = true;
		try {
			await api.post('/api/v1/auth/magic-link', { body: { email: value } });
		} catch (cause) {
			if (cause instanceof ApiError && cause.status === 422) {
				await fail('Please provide a valid email address');
				return;
			}
			pending = false;
			notifyFailure(cause);
			return;
		}
		authEmail.set(value);
		await goto('/auth/magic-link-sent');
	}
</script>

<form
	id="magic-link-form"
	method="POST"
	onsubmit={submit}
	aria-busy={pending || undefined}
	inert={pending}
>
	<Field.Group>
		<Field.Field>
			<Field.Label for="email">Email</Field.Label>
			<Input
				bind:ref={input}
				bind:value={email}
				id="email"
				name="email"
				type="email"
				placeholder="name@example.com"
				autocomplete="email"
				aria-invalid={message ? 'true' : 'false'}
				aria-describedby={message ? 'email-error' : undefined}
			/>
			{#if message}
				<Field.Error id="email-error">{message}</Field.Error>
			{/if}
		</Field.Field>
		<Button type="submit" class="w-full">Continue with Email</Button>
	</Field.Group>
</form>
