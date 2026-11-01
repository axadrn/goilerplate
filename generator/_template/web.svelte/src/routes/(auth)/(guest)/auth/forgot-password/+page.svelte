<script lang="ts">
	import { goto } from '$app/navigation';
	import { tick } from 'svelte';
	import { api, ApiError } from '#lib/api.js';
	import { authEmail } from '#lib/auth-email.js';
	import AuthCard from '#lib/components/auth/auth-card.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { notifyFailure } from '#lib/notify.js';

	let email = $state('');
	let message = $state('');
	let pending = $state(false);
	let input = $state<HTMLInputElement | null>(null);

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
			await api.post('/api/v1/auth/forgot-password', { body: { email: value } });
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
		await goto('/auth/forgot-password-sent');
	}
</script>

<AuthCard title="Forgot password?" description="We'll email you a secure reset link">
	<form
		id="forgot-password-form"
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
					placeholder="you@example.com"
					autocomplete="email"
					aria-invalid={message ? 'true' : 'false'}
					aria-describedby={message ? 'email-error' : undefined}
				/>
				{#if message}
					<Field.Error id="email-error">{message}</Field.Error>
				{/if}
			</Field.Field>
			<Button type="submit" class="w-full">Send reset link</Button>
			<Button href="/auth" variant="link" class="w-full">Back to sign in</Button>
		</Field.Group>
	</form>
</AuthCard>
